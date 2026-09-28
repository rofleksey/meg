package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"
)

const (
	helperTestTimeout = 10 * time.Second
	helperOutputSize  = 1024 * 1024
)

func TestRunnerDrainsOutput(t *testing.T) {
	request := helperRequest(t, "emit", strconv.Itoa(helperOutputSize), "0")
	request.OutputLimit = helperOutputSize

	for range 4 {
		result, err := (Runner{}).Run(t.Context(), request)
		if err != nil {
			t.Fatal(err)
		}

		if result.ExitCode != 0 || result.Truncated {
			t.Fatalf("result = %+v", result)
		}

		if !bytes.Equal(result.Stdout, bytes.Repeat([]byte("o"), helperOutputSize)) ||
			!bytes.Equal(result.Stderr, bytes.Repeat([]byte("e"), helperOutputSize)) {
			t.Fatalf("incomplete output: stdout=%d, stderr=%d", len(result.Stdout), len(result.Stderr))
		}
	}
}

func TestRunnerReportsExitAndOutputLimits(t *testing.T) {
	tests := []struct {
		name      string
		limit     int
		exit      int
		truncated bool
	}{
		{name: "exact fit", limit: 64},
		{name: "exit error", limit: 64, exit: 7},
		{name: "truncated", limit: 32, truncated: true},
		{name: "truncated exit error", limit: 32, exit: 7, truncated: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := helperRequest(t, "emit", "64", strconv.Itoa(test.exit))
			request.OutputLimit = test.limit
			result, err := (Runner{}).Run(t.Context(), request)

			if result.ExitCode != test.exit || result.Truncated != test.truncated {
				t.Fatalf("status=%d, truncated=%t", result.ExitCode, result.Truncated)
			}

			if !bytes.Equal(result.Stdout, bytes.Repeat([]byte("o"), test.limit)) ||
				!bytes.Equal(result.Stderr, bytes.Repeat([]byte("e"), test.limit)) {
				t.Fatalf("output = %q / %q", result.Stdout, result.Stderr)
			}

			if errors.Is(err, ErrOutputLimit) != test.truncated {
				t.Fatalf("error = %v, truncated = %t", err, test.truncated)
			}

			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) != (test.exit != 0) {
				t.Fatalf("error = %v, exit = %d", err, test.exit)
			}

			if test.exit == 0 && !test.truncated && err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRunnerPreservesArguments(t *testing.T) {
	argument := "spaces ; $(touch never) ' \" *"

	result, err := (Runner{}).Run(t.Context(), helperRequest(t, "argument", argument))
	if err != nil {
		t.Fatal(err)
	}

	if string(result.Stdout) != argument {
		t.Fatalf("argument = %q, want %q", result.Stdout, argument)
	}
}

func TestRunnerCanceledBeforeStart(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	result, err := (Runner{}).Run(ctx, helperRequest(t, "ready", marker))
	if !errors.Is(err, context.Canceled) || result.ExitCode != -1 {
		t.Fatalf("exit=%d, error=%v", result.ExitCode, err)
	}

	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled request started child: %v", err)
	}
}

func TestRunnerCancellationPreservesTruncation(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	request := helperRequest(t, "ready", marker)
	request.OutputLimit = 4
	finished := make(chan processOutcome, 1)

	go func() {
		result, err := (Runner{}).Run(ctx, request)
		finished <- processOutcome{result: result, err: err}
	}()

	waitForMarker(t, marker)
	cancel()

	outcome := awaitOutcome(t, finished)

	if !errors.Is(outcome.err, context.Canceled) || !errors.Is(outcome.err, ErrOutputLimit) {
		t.Fatalf("error = %v, want cancellation and output limit", outcome.err)
	}

	if string(outcome.result.Stdout) != "read" || !outcome.result.Truncated {
		t.Fatalf("result = %+v", outcome.result)
	}
}

func TestRunnerTimeoutTerminatesChild(t *testing.T) {
	request := helperRequest(t, "sleep")
	request.Timeout = 50 * time.Millisecond
	started := time.Now()
	result, err := (Runner{}).Run(t.Context(), request)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}

	if result.ExitCode == 0 {
		t.Fatal("timed-out child unexpectedly exited successfully")
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && result.ExitCode != exitErr.ExitCode() {
		t.Fatalf("exit code = %d, underlying process status = %d", result.ExitCode, exitErr.ExitCode())
	}

	if elapsed := time.Since(started); elapsed > processExitGrace+time.Second {
		t.Fatalf("timeout returned after %s", elapsed)
	}
}

func TestRunnerStartFailure(t *testing.T) {
	request := helperRequest(t, "sleep")
	request.Executable = filepath.Join(request.Directory, "missing-executable")
	result, err := (Runner{}).Run(t.Context(), request)

	if err == nil || result.ExitCode != -1 || len(result.Stdout) != 0 || len(result.Stderr) != 0 {
		t.Fatalf("result = %+v, error = %v", result, err)
	}
}

func TestRunnerInvalidRequest(t *testing.T) {
	result, err := (Runner{}).Run(t.Context(), Request{})
	if !errors.Is(err, ErrInvalidRequest) || result.ExitCode != -1 {
		t.Fatalf("result = %+v, error = %v", result, err)
	}
}

func TestLimitedBufferEmptyWriteAtLimit(t *testing.T) {
	buffer := limitedBuffer{limit: 1}
	if _, err := buffer.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}

	if count, err := buffer.Write(nil); count != 0 || err != nil || buffer.Truncated() {
		t.Fatalf("empty write = %d, %v; truncated=%t", count, err, buffer.Truncated())
	}
}

type processOutcome struct {
	result Result
	err    error
}

func helperRequest(t *testing.T, arguments ...string) Request {
	t.Helper()
	// Child helpers exit deliberately. The race runtime's extra shutdown sleep
	// must not become part of the runner's process-cleanup timing assertions.
	t.Setenv("GORACE", os.Getenv("GORACE")+" atexit_sleep_ms=0")

	return Request{
		Executable:  os.Args[0],
		Arguments:   append([]string{"-test.run=^TestProcessHelper$", "--"}, arguments...),
		Directory:   t.TempDir(),
		Timeout:     helperTestTimeout,
		OutputLimit: 1024,
	}
}

func waitForMarker(t *testing.T, path string) {
	t.Helper()

	timer := time.NewTimer(helperTestTimeout / 2)
	defer timer.Stop()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if _, err := os.Stat(path); err == nil {
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}

		select {
		case <-timer.C:
			t.Fatal("child did not become ready")
		case <-ticker.C:
		}
	}
}

func awaitOutcome(t *testing.T, finished <-chan processOutcome) processOutcome {
	t.Helper()

	select {
	case outcome := <-finished:
		return outcome
	case <-time.After(processExitGrace + time.Second):
		t.Fatal("process cleanup did not finish within its grace period")
		return processOutcome{}
	}
}

func TestProcessHelper(t *testing.T) {
	separator := slices.Index(os.Args, "--")
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}

	arguments := os.Args[separator+1:]

	switch arguments[0] {
	case "emit":
		count, err := strconv.Atoi(arguments[1])
		if err != nil {
			t.Fatal(err)
		}

		status, err := strconv.Atoi(arguments[2])
		if err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stdout.Write(bytes.Repeat([]byte("o"), count)); err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stderr.Write(bytes.Repeat([]byte("e"), count)); err != nil {
			t.Fatal(err)
		}

		os.Exit(status)
	case "argument":
		if _, err := fmt.Fprint(os.Stdout, arguments[1]); err != nil {
			t.Fatal(err)
		}

		os.Exit(0)
	case "ready":
		if _, err := fmt.Fprint(os.Stdout, "ready for cancellation"); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(arguments[1], nil, 0o600); err != nil {
			t.Fatal(err)
		}

		time.Sleep(2 * helperTestTimeout)
		os.Exit(0)
	case "sleep":
		time.Sleep(2 * helperTestTimeout)
		os.Exit(0)
	default:
		t.Fatalf("unknown helper mode: %s", arguments[0])
	}
}
