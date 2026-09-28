//go:build linux

package process

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	inheritedOutputMode = "inherited-output"
	escapedOutputMode   = "escaped-output"
)

func TestRunnerCleansUpProcessGroup(t *testing.T) {
	tests := []struct {
		mode      string
		cancel    bool
		waitDelay bool
		childExit int
		quickExit bool
		escaped   bool
	}{
		{mode: inheritedOutputMode, childExit: 0, quickExit: true},
		{mode: escapedOutputMode, waitDelay: true, childExit: 0, escaped: true},
		{mode: "parent-exits-on-term", cancel: true, childExit: -1, quickExit: true},
		{mode: "ignore-term", cancel: true, childExit: -1},
	}

	for _, test := range tests {
		t.Run(test.mode, func(t *testing.T) {
			var descendant *os.Process

			pidPath := filepath.Join(t.TempDir(), "descendant.pid")
			ctx, cancel := context.WithCancel(t.Context())
			// Register cleanup before starting either helper so a failing assertion
			// cannot leave a running descendant. Helpers also have a fallback timeout.
			t.Cleanup(func() {
				cancel()

				if descendant != nil {
					_ = descendant.Kill()
					_ = descendant.Release()
				}
			})

			request := helperRequest(t)
			request.Arguments = []string{"-test.run=^TestUnixProcessHelper$", "--", test.mode, pidPath}
			finished := make(chan processOutcome, 1)

			go func() {
				result, err := (Runner{}).Run(ctx, request)
				finished <- processOutcome{result: result, err: err}
			}()

			waitForMarker(t, pidPath)

			contents, err := os.ReadFile(pidPath)
			if err != nil {
				t.Fatal(err)
			}

			pid, err := strconv.Atoi(string(contents))
			if err != nil {
				t.Fatal(err)
			}

			// The helper parent waits for this acknowledgment before exiting. Open
			// an owned handle now, while its descendant is still alive, for cleanup
			// without signaling a numeric PID that could later be reused.
			descendant, err = os.FindProcess(pid)
			if err != nil {
				t.Fatal(err)
			}

			if err := os.WriteFile(pidPath+".continue", nil, 0o600); err != nil {
				t.Fatal(err)
			}

			started := time.Now()

			if test.cancel {
				cancel()
			}

			outcome := awaitOutcome(t, finished)
			if errors.Is(outcome.err, context.Canceled) != test.cancel ||
				errors.Is(outcome.err, exec.ErrWaitDelay) != test.waitDelay {
				t.Fatalf("error = %v", outcome.err)
			}

			if outcome.result.ExitCode != test.childExit {
				t.Fatalf("exit = %d, want %d", outcome.result.ExitCode, test.childExit)
			}

			if test.quickExit && time.Since(started) >= processExitGrace {
				t.Fatal("exited parent unnecessarily waited for full grace period")
			}

			if !test.escaped {
				waitForProcessDeath(t, pid)
			}
		})
	}
}

func TestSignalGroupWithoutProcess(t *testing.T) {
	if err := signalGroup(&exec.Cmd{}, syscall.SIGTERM); !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("signal = %v", err)
	}
}

func waitForProcessDeath(t *testing.T, pid int) {
	t.Helper()

	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		contents, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH) {
			return
		}

		if err != nil {
			t.Fatal(err)
		}

		// An orphan may remain a zombie until the host's init reaps it; it is
		// already dead and cannot retain pipes or perform further work.
		_, state, _ := strings.Cut(string(contents), ") ")
		if strings.HasPrefix(state, "Z ") {
			return
		}

		select {
		case <-deadline.C:
			t.Fatalf("descendant %d survived group cleanup", pid)
		case <-ticker.C:
		}
	}
}

func TestUnixProcessHelper(t *testing.T) {
	separator := slices.Index(os.Args, "--")
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}

	arguments := os.Args[separator+1:]
	if arguments[0] == "descendant" {
		signal.Ignore(syscall.SIGTERM)

		// Rename exposes the ready marker only after the complete PID is written.
		if err := os.WriteFile(arguments[1]+".tmp", []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
			t.Fatal(err)
		}

		if err := os.Rename(arguments[1]+".tmp", arguments[1]); err != nil {
			t.Fatal(err)
		}

		time.Sleep(2 * helperTestTimeout)
		os.Exit(0)
	}

	if arguments[0] == "ignore-term" {
		signal.Ignore(syscall.SIGTERM)
	}

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestUnixProcessHelper$", "--", "descendant", arguments[1])
	if arguments[0] == inheritedOutputMode || arguments[0] == escapedOutputMode {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if arguments[0] == escapedOutputMode {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	waitForMarker(t, arguments[1])
	waitForMarker(t, arguments[1]+".continue")

	if arguments[0] == inheritedOutputMode || arguments[0] == escapedOutputMode {
		os.Exit(0)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}

	os.Exit(0)
}
