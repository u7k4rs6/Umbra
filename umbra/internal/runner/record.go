package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Recording is the on-disk form of a scenario: every call a real run made,
// in order, with its output. Replaying one lets a test exercise the whole
// pipeline with no Entire, no git and no agent.
type Recording struct {
	Scenario string  `json:"scenario"`
	Entries  []Entry `json:"entries"`
	Notes    string  `json:"notes,omitempty"`
}

// Entry pairs one call with what it produced.
type Entry struct {
	Call   Call   `json:"call"`
	Result Result `json:"result"`
}

// Scrubber rewrites text before it is written to a recording. It removes
// absolute paths, user names and token shapes, per SECURITY_AND_ACCESS.md.
type Scrubber func(string) string

// Recorder wraps a Runner and stores everything it sees.
type Recorder struct {
	Inner   Runner
	Scrub   Scrubber
	mu      sync.Mutex
	entries []Entry
}

// NewRecorder wraps inner. A nil scrub function records text unchanged, which
// is only appropriate for a throwaway run.
func NewRecorder(inner Runner, scrub Scrubber) *Recorder {
	return &Recorder{Inner: inner, Scrub: scrub}
}

// Run delegates to the inner runner and stores the result.
func (r *Recorder) Run(ctx context.Context, name string, args []string, stdin []byte) ([]byte, []byte, int, error) {
	stdout, stderr, exit, err := r.Inner.Run(ctx, name, args, stdin)

	scrub := r.Scrub
	if scrub == nil {
		scrub = func(s string) string { return s }
	}
	entry := Entry{
		Call: Call{Name: name, Args: scrubArgs(scrub, args)},
		Result: Result{
			Stdout: scrub(string(stdout)),
			Stderr: scrub(string(stderr)),
			Exit:   exit,
		},
	}
	if err != nil {
		entry.Result.Err = scrub(err.Error())
	}

	r.mu.Lock()
	r.entries = append(r.entries, entry)
	r.mu.Unlock()

	return stdout, stderr, exit, err
}

// Save writes the recording into dir as recording.json.
func (r *Recorder) Save(dir, scenario, notes string) error {
	r.mu.Lock()
	entries := append([]Entry(nil), r.entries...)
	r.mu.Unlock()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	rec := Recording{Scenario: scenario, Entries: entries, Notes: notes}
	blob, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	blob = append(blob, '\n')
	return os.WriteFile(filepath.Join(dir, "recording.json"), blob, 0o644)
}

func scrubArgs(scrub Scrubber, args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = scrub(a)
	}
	return out
}

// Replay serves recorded results and never starts a process.
type Replay struct {
	byKey map[string][]Result
	seen  map[string]int
	order []Call
	mu    sync.Mutex
	// Strict makes an unknown call an error. When false an unknown call
	// returns empty output and exit 0, which is how a scenario that does not
	// exercise a step stays green.
	Strict bool
}

// LoadRecording reads a recording directory written by Recorder.Save.
func LoadRecording(dir string) (*Recording, error) {
	blob, err := os.ReadFile(filepath.Join(dir, "recording.json"))
	if err != nil {
		return nil, err
	}
	var rec Recording
	if err := json.Unmarshal(blob, &rec); err != nil {
		return nil, fmt.Errorf("recording %s: %w", dir, err)
	}
	return &rec, nil
}

// NewReplay builds a replaying runner from a recording.
func NewReplay(rec *Recording) *Replay {
	r := &Replay{byKey: map[string][]Result{}, seen: map[string]int{}, Strict: true}
	for _, e := range rec.Entries {
		k := e.Call.Key()
		r.byKey[k] = append(r.byKey[k], e.Result)
		r.order = append(r.order, e.Call)
	}
	return r
}

// Run returns the next recorded result for this call.
//
// Repeated identical calls are served in the order they were recorded, so a
// command run twice with different output (a baseline and then a verify)
// replays faithfully. Once a call's results are exhausted the last one
// repeats, which keeps a retry loop from falling off the end.
func (r *Replay) Run(ctx context.Context, name string, args []string, stdin []byte) ([]byte, []byte, int, error) {
	call := Call{Name: name, Args: args}
	k := call.Key()

	r.mu.Lock()
	defer r.mu.Unlock()

	results, ok := r.byKey[k]
	if !ok || len(results) == 0 {
		if r.Strict {
			return nil, nil, 0, fmt.Errorf("no recorded result for: %s", call.String())
		}
		return nil, nil, 0, nil
	}
	i := r.seen[k]
	if i >= len(results) {
		i = len(results) - 1
	}
	r.seen[k] = i + 1
	res := results[i]

	var err error
	if res.Err != "" {
		err = fmt.Errorf("%s", res.Err)
	}
	return []byte(res.Stdout), []byte(res.Stderr), res.Exit, err
}

// Calls lists the recorded calls in order, for assertions in tests.
func (r *Replay) Calls() []Call { return append([]Call(nil), r.order...) }

// Keys lists the recorded call keys, sorted, for diagnostics.
func (r *Replay) Keys() []string {
	out := make([]string, 0, len(r.byKey))
	for k := range r.byKey {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Fake is a hand-built runner for unit tests that do not need a full
// recording. Responses are keyed the same way as a replay.
type Fake struct {
	Responses map[string]Result
	Calls     []Call
	// Default is returned for an unknown call.
	Default Result
}

// NewFake builds an empty fake.
func NewFake() *Fake { return &Fake{Responses: map[string]Result{}} }

// Set registers a response for one call.
func (f *Fake) Set(name string, args []string, res Result) {
	f.Responses[Call{Name: name, Args: args}.Key()] = res
}

// Run records the call and returns the registered response.
func (f *Fake) Run(ctx context.Context, name string, args []string, stdin []byte) ([]byte, []byte, int, error) {
	call := Call{Name: name, Args: append([]string(nil), args...)}
	f.Calls = append(f.Calls, call)
	res, ok := f.Responses[call.Key()]
	if !ok {
		res = f.Default
	}
	var err error
	if res.Err != "" {
		err = fmt.Errorf("%s", res.Err)
	}
	return []byte(res.Stdout), []byte(res.Stderr), res.Exit, err
}

// readFile is a small indirection so tests can read a recording's bytes
// without importing the operating system package themselves.
func readFile(path string) ([]byte, error) { return os.ReadFile(path) }
