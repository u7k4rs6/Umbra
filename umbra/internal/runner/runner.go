// Package runner is the single boundary between Umbra and every external
// process it starts. Nothing else in Umbra may call exec. Tests replace the
// real runner with a replaying fake so no test needs Entire, git or an agent.
package runner

import (
	"context"
	"fmt"
	"strings"
)

// Runner starts one external process and returns its output.
//
// Callers pass argv as separate elements. There is no shell anywhere in
// Umbra, so a symbol name or a path that happens to contain shell syntax is
// inert.
type Runner interface {
	Run(ctx context.Context, name string, args []string, stdin []byte) (stdout, stderr []byte, exit int, err error)
}

// Call is one invocation, used as the key of a recording.
type Call struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}

// Key is the stable identity of a call within a recording.
func (c Call) Key() string {
	return c.Name + "\x00" + strings.Join(c.Args, "\x00")
}

// String renders the call the way the report echoes it to the reader.
func (c Call) String() string {
	parts := make([]string, 0, len(c.Args)+1)
	parts = append(parts, c.Name)
	parts = append(parts, c.Args...)
	return strings.Join(parts, " ")
}

// Result is what one invocation produced.
type Result struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Exit   int    `json:"exit"`
	Err    string `json:"err,omitempty"`
}

// ExitError reports a non-zero exit from a process that was expected to
// succeed. Callers that tolerate a non-zero exit read the exit code instead.
type ExitError struct {
	Call   Call
	Exit   int
	Stderr string
}

func (e *ExitError) Error() string {
	msg := strings.TrimSpace(e.Stderr)
	if msg == "" {
		return fmt.Sprintf("%s exited %d", e.Call.String(), e.Exit)
	}
	if len(msg) > 300 {
		msg = msg[:300] + "..."
	}
	return fmt.Sprintf("%s exited %d: %s", e.Call.String(), e.Exit, msg)
}
