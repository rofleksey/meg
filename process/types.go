// Package process runs trusted executable requests without invoking a shell.
package process

import (
	"errors"
	"sync"
	"time"
)

const processExitGrace = 2 * time.Second

var ErrOutputLimit = errors.New("process output limit reached")

// Request must be constructed from trusted local application policy.
type Request struct {
	Executable  string
	Arguments   []string
	Directory   string
	Timeout     time.Duration
	OutputLimit int
}

type Result struct {
	Stdout    []byte
	Stderr    []byte
	Truncated bool
	ExitCode  int
}

type Runner struct{}

type limitedBuffer struct {
	mu        sync.Mutex
	bytes     []byte
	limit     int
	truncated bool
}
