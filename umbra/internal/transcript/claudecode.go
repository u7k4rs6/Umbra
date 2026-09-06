package transcript

import (
	"bufio"
	"bytes"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// The record shapes below are the ones the Step 0 probe captured from
// `entire checkpoint explain <id> --raw-transcript`. Every field the adapter
// depends on was confirmed present: Read carries file_path and, for a partial
// read, offset and limit; Edit carries file_path; every record carries cwd,
// timestamp and sessionId; tool_result content is a string, and the array of
// blocks form is accepted too because the format documents both.

type record struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	CWD       string          `json:"cwd"`
	SessionID string          `json:"sessionId"`
	Message   json.RawMessage `json:"message"`
}

type message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type block struct {
	Type string `json:"type"`
	Text string `json:"text"`

	// tool_use
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`

	// tool_result
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

type readInput struct {
	FilePath string `json:"file_path"`
	Offset   *int   `json:"offset"`
	Limit    *int   `json:"limit"`
}

type pathInput struct {
	FilePath     string `json:"file_path"`
	NotebookPath string `json:"notebook_path"`
	Path         string `json:"path"`
	Pattern      string `json:"pattern"`
	Command      string `json:"command"`
}

// ClaudeCode parses Claude Code JSONL into a Session.
type ClaudeCode struct {
	// RepoRoot makes absolute paths repository relative. The transcript stores
	// absolute paths, which the probe confirmed.
	RepoRoot string
}

// Name is the adapter name printed in the report inputs.
func (ClaudeCode) Name() string { return "claude-code" }

// Parse reads the JSONL and returns the normalized session.
//
// A line that does not parse is skipped rather than failing the run: a
// truncated or partly unknown transcript still yields the evidence it does
// carry, and missing evidence becomes the unknown state downstream.
func (a ClaudeCode) Parse(jsonl []byte) (*Session, error) {
	s := &Session{Agent: a.Name()}
	seq := 0
	next := func() int { seq++; return seq }

	// Tool uses awaiting their result, so a Grep can take the paths its
	// result listed and a Read can be linked to the content it returned.
	pending := map[string]*Event{}
	sessions := map[string]bool{}

	sc := bufio.NewScanner(bytes.NewReader(jsonl))
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)

	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var rec record
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}
		if rec.SessionID != "" {
			sessions[rec.SessionID] = true
		}
		ts := parseTime(rec.Timestamp)
		root := a.RepoRoot
		if root == "" {
			root = rec.CWD
		}

		var msg message
		if len(rec.Message) > 0 {
			_ = json.Unmarshal(rec.Message, &msg)
		}
		blocks := decodeBlocks(msg.Content)

		switch rec.Type {
		case "assistant":
			for _, b := range blocks {
				switch b.Type {
				case "text":
					if strings.TrimSpace(b.Text) == "" {
						continue
					}
					n := next()
					s.Events = append(s.Events, Event{Seq: n, TS: ts, Kind: AssistantText})
					s.texts = append(s.texts, textAt{seq: n, text: b.Text})
				case "tool_use":
					if ev := a.toolUse(b, root, ts, next); ev != nil {
						s.Events = append(s.Events, *ev)
						if ev.ToolID != "" {
							pending[ev.ToolID] = &s.Events[len(s.Events)-1]
						}
					}
				}
			}
		case "user":
			// A user record is either a prompt or the results of tool calls.
			sawResult := false
			for _, b := range blocks {
				if b.Type != "tool_result" {
					continue
				}
				sawResult = true
				text := resultText(b.Content)
				if text == "" {
					continue
				}
				owner := pending[b.ToolUseID]
				paths := extractPaths(text, root)

				if owner != nil && (owner.Kind == Grep || owner.Kind == Glob) {
					owner.Paths = paths
					continue
				}
				// Any other result that quotes paths is partial evidence: the
				// file's content passed in front of the session without it
				// opening the file. This is the quoted tier.
				if len(paths) > 0 {
					s.Events = append(s.Events, Event{
						Seq: next(), TS: ts, Kind: ResultFile, Paths: paths,
					})
				}
			}
			if !sawResult && len(blocks) > 0 {
				s.Events = append(s.Events, Event{Seq: next(), TS: ts, Kind: Prompt})
			} else if !sawResult && len(rec.Message) > 0 {
				s.Events = append(s.Events, Event{Seq: next(), TS: ts, Kind: Prompt})
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	for id := range sessions {
		s.SessionIDs = append(s.SessionIDs, id)
	}
	sort.Strings(s.SessionIDs)
	return s, nil
}

func (a ClaudeCode) toolUse(b block, root string, ts time.Time, next func() int) *Event {
	switch b.Name {
	case "Read":
		var in readInput
		_ = json.Unmarshal(b.Input, &in)
		if in.FilePath == "" {
			return nil
		}
		path := rel(root, in.FilePath)
		if path == "" {
			return nil
		}
		ev := Event{Seq: next(), TS: ts, Kind: Read, Path: path, ToolID: b.ID}
		// offset and limit are present only for a partial read, which is what
		// makes the glance tier possible. offset is 1 based.
		if in.Offset != nil && in.Limit != nil {
			start := *in.Offset
			if start < 1 {
				start = 1
			}
			end := start + *in.Limit - 1
			ev.Range = &[2]int{start, end}
		}
		return &ev

	case "Grep":
		var in pathInput
		_ = json.Unmarshal(b.Input, &in)
		return &Event{Seq: next(), TS: ts, Kind: Grep, ToolID: b.ID}

	case "Glob":
		var in pathInput
		_ = json.Unmarshal(b.Input, &in)
		return &Event{Seq: next(), TS: ts, Kind: Glob, ToolID: b.ID}

	case "Edit", "Write", "MultiEdit", "NotebookEdit":
		var in pathInput
		_ = json.Unmarshal(b.Input, &in)
		p := in.FilePath
		if p == "" {
			p = in.NotebookPath
		}
		if p == "" {
			return nil
		}
		rp := rel(root, p)
		if rp == "" {
			return nil
		}
		return &Event{Seq: next(), TS: ts, Kind: Edit, Path: rp, ToolID: b.ID}

	case "Bash":
		var in pathInput
		_ = json.Unmarshal(b.Input, &in)
		// The command is kept for the timeline only. Umbra never runs anything
		// it found in a transcript.
		return &Event{Seq: next(), TS: ts, Kind: Command, Cmd: in.Command, ToolID: b.ID}
	}
	return nil
}

// decodeBlocks accepts both content shapes: an array of blocks, and a bare
// string, which is how a plain user prompt is stored.
func decodeBlocks(raw json.RawMessage) []block {
	if len(raw) == 0 {
		return nil
	}
	var blocks []block
	if err := json.Unmarshal(raw, &blocks); err == nil {
		return blocks
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil && s != "" {
		return []block{{Type: "text", Text: s}}
	}
	return nil
}

// resultText flattens tool_result content, which the probe saw as a plain
// string and which the format also permits as an array of blocks.
func resultText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []block
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var sb strings.Builder
		for _, b := range blocks {
			if b.Text != "" {
				sb.WriteString(b.Text)
				sb.WriteByte('\n')
			}
		}
		return sb.String()
	}
	return ""
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// rel makes an absolute path repository relative.
//
// A path outside the repository returns the empty string and the caller drops
// it. It can never match a symbol in the field, so it is noise, and carrying
// it into a report would leak the shape of the machine the session ran on:
// a search that happened to list files in another project would put those
// paths in the timeline. Only when the repository root is unknown is an
// absolute path passed through, because then nothing can be judged.
func rel(root, p string) string {
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)
	if !filepath.IsAbs(p) {
		return filepath.ToSlash(strings.TrimPrefix(p, "./"))
	}
	if root == "" {
		return filepath.ToSlash(p)
	}
	r, err := filepath.Rel(filepath.Clean(root), p)
	if err != nil || r == ".." || strings.HasPrefix(r, "../") {
		return ""
	}
	return filepath.ToSlash(r)
}
