// Package process runs trusted executable requests without invoking a shell.
package process

import (
	"errors"
	"time"
)

const processExitGrace = 2 * time.Second

var (
	ErrInvalidRequest = errors.New("invalid process request")
	ErrOutputLimit    = errors.New("process output limit reached")
)

// Request must be constructed from trusted local application policy.
type Request struct {
	Executable string
	Arguments  []string
	Directory  string
	Timeout    time.Duration
	// OutputLimit is the maximum number of bytes retained per output stream.
	OutputLimit int
}

type Result struct {
	Stdout    []byte
	Stderr    []byte
	Truncated bool
	// ExitCode is the direct child's observed exit status, or -1 if it did not
	// start or was terminated by a signal. A zero status does not imply no error:
	// output can be truncated or a descendant can prevent output draining.
	ExitCode int
}

type Runner struct{}

type limitedBuffer struct {
	bytes     []byte
	limit     int
	truncated bool
}
