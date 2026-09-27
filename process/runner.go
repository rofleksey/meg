package process

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"
)

// Run captures bounded stdout and stderr and waits for process-tree cleanup
// before returning on cancellation.
func (Runner) Run(ctx context.Context, request Request) (Result, error) {
	if request.Executable == "" || request.Directory == "" || request.Timeout <= 0 || request.OutputLimit <= 0 {
		return Result{}, errors.New("invalid process request")
	}
	operationCtx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()

	// CommandContext would kill only the direct child before callers can
	// perform action-specific cleanup, so lifecycle control stays here.
	cmd := exec.Command(request.Executable, request.Arguments...)
	cmd.Dir = request.Directory
	configureProcess(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return Result{}, err
	}
	if err := cmd.Start(); err != nil {
		return Result{}, err
	}

	out := limitedBuffer{limit: request.OutputLimit}
	errOut := limitedBuffer{limit: request.OutputLimit}
	var readers sync.WaitGroup
	readers.Add(2)
	go func() { defer readers.Done(); _, _ = io.Copy(&out, stdout) }()
	go func() { defer readers.Done(); _, _ = io.Copy(&errOut, stderr) }()
	finished := make(chan error, 1)
	go func() { finished <- cmd.Wait() }()

	var runErr error
	select {
	case runErr = <-finished:
	case <-operationCtx.Done():
		_ = interruptProcess(cmd)
		select {
		case <-finished:
		case <-time.After(processExitGrace):
			_ = killProcess(cmd)
			<-finished
		}
		runErr = operationCtx.Err()
	}
	readers.Wait()
	result := Result{Stdout: out.Bytes(), Stderr: errOut.Bytes(), Truncated: out.Truncated() || errOut.Truncated(), ExitCode: exitCode(runErr)}
	if result.Truncated {
		return result, ErrOutputLimit
	}
	return result, runErr
}

func (b *limitedBuffer) Write(value []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - len(b.bytes)
	if remaining <= 0 {
		b.truncated = true
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
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.bytes...)
}

func (b *limitedBuffer) Truncated() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.truncated
}
