package process

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"
)

func TestRunnerTimeoutTerminatesChild(t *testing.T) {
	started := time.Now()

	result, err := (Runner{}).Run(context.Background(), Request{
		Executable:  os.Args[0],
		Arguments:   []string{"-test.run=TestProcessHelper", "--", "sleep"},
		Directory:   t.TempDir(),
		Timeout:     50 * time.Millisecond,
		OutputLimit: 1024,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}

	if result.ExitCode != -1 {
		t.Fatalf("exit code = %d, want -1 for timeout", result.ExitCode)
	}

	if elapsed := time.Since(started); elapsed > processExitGrace+time.Second {
		t.Fatalf("timeout returned after %s", elapsed)
	}
}

func TestProcessHelper(t *testing.T) {
	separator := slices.Index(os.Args, "--")
	if separator < 0 || separator+1 >= len(os.Args) || os.Args[separator+1] != "sleep" {
		return
	}

	for {
		time.Sleep(time.Hour)
	}
}
