//go:build windows

package process

import "os/exec"

func configureProcess(*exec.Cmd) {}

func interruptProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

func killProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
