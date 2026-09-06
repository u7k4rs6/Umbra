package runner

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Timeouts from SECURITY_AND_ACCESS.md. A caller may override per call by
// passing a context with an earlier deadline.
const (
	GraphTimeout   = 60 * time.Second
	VerifyTimeout  = 10 * time.Minute
	SweepTimeout   = 20 * time.Minute
	SnapshotWarnAt = 30 * time.Second
	DefaultTimeout = 2 * time.Minute
)

// Exec runs real processes. It is the only implementation that touches the
// operating system.
type Exec struct {
	// Dir is the working directory for every call. Empty means inherit.
	Dir string
	// Env replaces the environment when non-nil.
	Env []string

	mu    sync.Mutex
	calls []Call
}

// NewExec builds a runner rooted at dir.
func NewExec(dir string) *Exec { return &Exec{Dir: dir} }

// Run starts name with args and waits for it to finish.
func (e *Exec) Run(ctx context.Context, name string, args []string, stdin []byte) ([]byte, []byte, int, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}

	e.mu.Lock()
	e.calls = append(e.calls, Call{Name: name, Args: append([]string(nil), args...)})
	e.mu.Unlock()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = e.Dir
	if e.Env != nil {
		cmd.Env = e.Env
	} else {
		cmd.Env = os.Environ()
	}
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb

	err := cmd.Run()
	exit := 0
	if err != nil {
		var ee *exec.ExitError
		if ok := asExitError(err, &ee); ok {
			exit = ee.ExitCode()
			err = nil
		}
	}
	return out.Bytes(), errb.Bytes(), exit, err
}

// Calls returns every invocation made so far, in order.
func (e *Exec) Calls() []Call {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]Call(nil), e.calls...)
}

func asExitError(err error, target **exec.ExitError) bool {
	ee, ok := err.(*exec.ExitError)
	if ok {
		*target = ee
	}
	return ok
}
