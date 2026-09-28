//go:build linux

package process

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

func runProcess(cmd *exec.Cmd) error {
	var groupMu sync.Mutex

	canSignalGroup := true
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		groupMu.Lock()
		defer groupMu.Unlock()

		if !canSignalGroup {
			return os.ErrProcessDone
		}

		return signalGroup(cmd, syscall.SIGTERM)
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Observe exit without reaping the leader. Its reserved PID pins the group
	// identity, so cleanup cannot signal a newly reused process-group ID.
	waitErr := waitForExit(cmd.Process.Pid)

	groupMu.Lock()
	canSignalGroup = false

	var cleanupErr error
	if waitErr == nil {
		cleanupErr = signalGroup(cmd, syscall.SIGKILL)
	} else {
		// If exit ownership could not be confirmed, use the owned process handle
		// only; an external reaper may have already released the numeric PID.
		cleanupErr = cmd.Process.Kill()
	}

	groupMu.Unlock()

	if errors.Is(cleanupErr, os.ErrProcessDone) {
		cleanupErr = nil
	} else if cleanupErr != nil {
		cleanupErr = fmt.Errorf("clean up process: %w", cleanupErr)
	}

	// Cancel is now prevented from using the raw group ID. It must not hold
	// groupMu while Wait reaps the leader and synchronizes with its watcher.
	return errors.Join(waitErr, cleanupErr, cmd.Wait())
}

func waitForExit(pid int) error {
	for {
		err := unix.Waitid(unix.P_PID, pid, nil, unix.WEXITED|unix.WNOWAIT, nil)
		if !errors.Is(err, syscall.EINTR) {
			return err
		}
	}
}

// signalGroup is called only while the unreaped leader still owns its PID.
func signalGroup(cmd *exec.Cmd, signal syscall.Signal) error {
	if cmd.Process == nil {
		return os.ErrProcessDone
	}

	err := syscall.Kill(-cmd.Process.Pid, signal)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}

	return err
}
