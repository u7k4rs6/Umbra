package checkpoint

import (
	"context"
	"fmt"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

// RawTranscript returns the stored JSONL for a checkpoint.
//
// Missing data is a state, never an error: a checkpoint with no transcript
// yields empty bytes and the classifier reports every node as unknown.
func RawTranscript(ctx context.Context, run runner.Runner, id string) ([]byte, error) {
	if id == "" {
		return nil, nil
	}
	args := []string{"checkpoint", "explain", id, "--raw-transcript"}
	stdout, _, exit, err := run.Run(ctx, "entire", args, nil)
	if err != nil {
		return nil, err
	}
	if exit != 0 {
		return nil, nil
	}
	return stdout, nil
}

// StoredSummary returns Entire's own summary sentence for a checkpoint, or
// the empty string when there is none.
//
// The probe found that imported checkpoints never carry one: `--short` prints
// a literal "No summary" line instead. Recognising that here is what lets the
// caller fall back to the last assistant sentence before the commit.
func StoredSummary(ctx context.Context, run runner.Runner, id string) (string, error) {
	if id == "" {
		return "", nil
	}
	args := []string{"checkpoint", "explain", id, "--short"}
	stdout, _, exit, err := run.Run(ctx, "entire", args, nil)
	if err != nil {
		return "", err
	}
	if exit != 0 {
		return "", nil
	}
	return ParseSummary(string(stdout)), nil
}

// ParseSummary pulls the text under the "## Summary" heading out of the
// `checkpoint explain --short` rendering.
func ParseSummary(text string) string {
	lines := strings.Split(text, "\n")
	idx := -1
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "## Summary" {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ""
	}
	var body []string
	for _, ln := range lines[idx+1:] {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "##") || strings.HasPrefix(t, "──") {
			break
		}
		if t == "" {
			if len(body) > 0 {
				break
			}
			continue
		}
		body = append(body, t)
	}
	joined := strings.TrimSpace(strings.Join(body, " "))

	// Entire renders the absent case in italics. Treat it as absent.
	if joined == "" || strings.HasPrefix(joined, "*No summary") {
		return ""
	}
	return joined
}

// Intent pulls the "## Intent" line, which is the user's own prompt as Entire
// stored it. It is never used as the session's account of itself; it is only
// available for the header when a report needs to say what was asked.
func Intent(text string) string {
	lines := strings.Split(text, "\n")
	idx := -1
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "## Intent" {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ""
	}
	for _, ln := range lines[idx+1:] {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "##") {
			return ""
		}
		return t
	}
	return ""
}

// Describe renders the resolution for the report header.
func (r *Resolution) Describe() string {
	id := r.CheckpointID
	if id == "" {
		id = "(none)"
	}
	agent := r.Agent
	if agent == "" {
		agent = "unknown agent"
	}
	return fmt.Sprintf("%s  %s  %s  via %s", id, short(r.Commit), agent, r.Route)
}
