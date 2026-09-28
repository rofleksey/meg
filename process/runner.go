package process

import (
	"context"
	"errors"
	"os/exec"
)

// Run captures up to OutputLimit bytes from each output stream. Excess output
// is discarded while the command continues; ErrOutputLimit is joined with any
// execution error, so errors.Is and errors.As can inspect both failures.
//
// Cancellation signals the process group on Linux and kills the direct child on
// other platforms. After a two-second grace period, the child is killed and
// inherited output pipes are closed. Remaining Linux process-group members are
// also killed after normal child exit. Descendants that escape the group, and
// descendants on other platforms, cannot be terminated by this runner.
func (Runner) Run(ctx context.Context, request Request) (Result, error) {
	if request.Executable == "" || request.Directory == "" || request.Timeout <= 0 || request.OutputLimit <= 0 {
		return Result{ExitCode: -1}, ErrInvalidRequest
	}

	operationCtx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()

	out := limitedBuffer{limit: request.OutputLimit}
	errOut := limitedBuffer{limit: request.OutputLimit}
	// Request contains trusted application policy, never a shell command string.
	//nolint:gosec // Executable and argv are deliberately supplied by the trusted caller.
	cmd := exec.CommandContext(operationCtx, request.Executable, request.Arguments...)
	cmd.Dir = request.Directory
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	// Cmd owns and drains the copy goroutines before Wait returns. WaitDelay
	// bounds that drain if a descendant retains stdout or stderr after exit.
	cmd.WaitDelay = processExitGrace
	runErr := runProcess(cmd)

	if err := operationCtx.Err(); err != nil && !errors.Is(runErr, err) {
		runErr = errors.Join(err, runErr)
	}

	result := Result{
		Stdout: out.Bytes(), Stderr: errOut.Bytes(),
		Truncated: out.Truncated() || errOut.Truncated(),
		ExitCode:  cmd.ProcessState.ExitCode(),
	}
	if result.Truncated {
		runErr = errors.Join(ErrOutputLimit, runErr)
	}

	return result, runErr
}

func (b *limitedBuffer) Write(value []byte) (int, error) {
	remaining := b.limit - len(b.bytes)
	if remaining <= 0 {
		b.truncated = b.truncated || len(value) > 0
		return len(value), nil
	}

	if len(value) > remaining {
		b.bytes = append(b.bytes, value[:remaining]...)
		b.truncated = true

		return len(value), nil
	}

	b.bytes = append(b.bytes, value...)

	return len(value), nil
}

func (b *limitedBuffer) Bytes() []byte {
	return append([]byte(nil), b.bytes...)
}

func (b *limitedBuffer) Truncated() bool {
	return b.truncated
}
