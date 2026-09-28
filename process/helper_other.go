//go:build !linux

package process

import "os/exec"

// Other platforms use CommandContext's owned process handle. Descendant
// termination requires application-owned platform infrastructure.
func runProcess(cmd *exec.Cmd) error {
	return cmd.Run()
}
