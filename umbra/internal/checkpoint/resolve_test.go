package checkpoint

import (
	"context"
	"testing"
	"time"

	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

func TestParseTrailer(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"present", "fix the thing\n\nEntire-Checkpoint: a1b2c3d4e5f6\n", "a1b2c3d4e5f6"},
		{"case is preserved so git log can grep the exact text", "x\n\nEntire-Checkpoint: A1B2C3D4E5F6\n", "A1B2C3D4E5F6"},
		{"absent", "make the tax rate explicit on compute_total\n\nChecked the callers.\n", ""},
		{"not at line start", "see Entire-Checkpoint: a1b2c3d4e5f6 inline\n", ""},
		{"too short", "x\n\nEntire-Checkpoint: abc\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ParseTrailer(c.body); got != c.want {
				t.Fatalf("ParseTrailer = %q, want %q", got, c.want)
			}
		})
	}
}

// The listing bytes below are the real output shape captured in the Step 0
// probe, including the warning Entire prints when the checkpoint remote is
// unreachable.
const probeListing = `[entire] Warning: could not reach checkpoint remote; showing local checkpoints only.
[
  {
    "checkpoint_id": "b20f84567474",
    "session_id": "8017734d-cdf6-4a18-b2f8-f4b5f637ec35",
    "agent": "Claude Code",
    "date": "2026-09-04T17:54:32.767Z",
    "message": "then make a repo named Umbra on my github...",
    "is_logs_only": true,
    "session_count": 1,
    "session_ids": ["8017734d-cdf6-4a18-b2f8-f4b5f637ec35"]
  },
  {
    "checkpoint_id": "eee22ab0fd95",
    "session_id": "8017734d-cdf6-4a18-b2f8-f4b5f637ec35",
    "agent": "Claude Code",
    "date": "2026-09-04T17:51:16.348Z",
    "message": "You are starting the build of Umbra...",
    "is_logs_only": true,
    "session_count": 1,
    "session_ids": ["8017734d-cdf6-4a18-b2f8-f4b5f637ec35"]
  }
]`

func TestParseListingSkipsWarningLine(t *testing.T) {
	got, err := ParseListing([]byte(probeListing))
	if err != nil {
		t.Fatalf("ParseListing: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d listings, want 2", len(got))
	}
	if got[0].CheckpointID != "b20f84567474" {
		t.Fatalf("first id = %q", got[0].CheckpointID)
	}
	if !got[0].IsLogsOnly {
		t.Fatal("expected is_logs_only true")
	}
	if got[0].Time().IsZero() {
		t.Fatal("expected a parseable date")
	}
}

func TestParseListingEmptyArray(t *testing.T) {
	got, err := ParseListing([]byte("[entire] Warning: nope\n[]\n"))
	if err != nil {
		t.Fatalf("ParseListing: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d, want 0", len(got))
	}
}

func newResolver(f *runner.Fake) *Resolver {
	r := New(f, "/repo")
	r.Now = func() time.Time { return time.Date(2026, 9, 4, 18, 0, 0, 0, time.UTC) }
	return r
}

func TestResolveByTrailer(t *testing.T) {
	f := runner.NewFake()
	f.Set("entire", []string{"checkpoint", "list", "--json"}, runner.Result{Stdout: probeListing})
	f.Set("git", []string{"-C", "/repo", "rev-parse", "--verify", "HEAD^{commit}"},
		runner.Result{Stdout: "00634433c2bc2ac6d1da8e76f45847e7f6702a36\n"})
	f.Set("git", []string{"-C", "/repo", "rev-list", "--parents", "-n", "1", "00634433c2bc2ac6d1da8e76f45847e7f6702a36"},
		runner.Result{Stdout: "00634433c2bc2ac6d1da8e76f45847e7f6702a36 faf4dc616ae391650ddd4680657d64b79d9d248f\n"})
	f.Set("git", []string{"-C", "/repo", "log", "-1", "--format=%B", "00634433c2bc2ac6d1da8e76f45847e7f6702a36"},
		runner.Result{Stdout: "make the tax rate explicit\n\nEntire-Checkpoint: b20f84567474\n"})

	res, err := newResolver(f).Resolve(context.Background(), "HEAD")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if res.Route != RouteTrailer {
		t.Fatalf("route = %q, want %q", res.Route, RouteTrailer)
	}
	if res.CheckpointID != "b20f84567474" {
		t.Fatalf("checkpoint = %q", res.CheckpointID)
	}
	if res.Parent != "faf4dc616ae391650ddd4680657d64b79d9d248f" {
		t.Fatalf("parent = %q", res.Parent)
	}
	if res.IsMerge {
		t.Fatal("expected a non-merge commit")
	}
	if res.Agent != "Claude Code" {
		t.Fatalf("agent = %q", res.Agent)
	}
}

// The probe's own commits carry no trailer, because the Claude Code hooks were
// installed part way through the session. Pairing by session time is the
// documented fallback for exactly that case.
func TestResolveBySessionWindowWhenNoTrailer(t *testing.T) {
	f := runner.NewFake()
	f.Set("entire", []string{"checkpoint", "list", "--json"}, runner.Result{Stdout: probeListing})
	f.Set("git", []string{"-C", "/repo", "rev-parse", "--verify", "HEAD^{commit}"},
		runner.Result{Stdout: "0063443\n"})
	f.Set("git", []string{"-C", "/repo", "rev-list", "--parents", "-n", "1", "0063443"},
		runner.Result{Stdout: "0063443 faf4dc6\n"})
	f.Set("git", []string{"-C", "/repo", "log", "-1", "--format=%B", "0063443"},
		runner.Result{Stdout: "make the tax rate explicit on compute_total\n\nChecked the callers.\n"})
	f.Set("git", []string{"-C", "/repo", "log", "-1", "--format=%cI", "0063443"},
		runner.Result{Stdout: "2026-09-04T17:58:00+00:00\n"})

	res, err := newResolver(f).Resolve(context.Background(), "HEAD")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if res.Route != RouteSessionWindow {
		t.Fatalf("route = %q, want %q", res.Route, RouteSessionWindow)
	}
	// 17:54 precedes the commit at 17:58; 17:51 is further away. The nearer
	// past checkpoint wins.
	if res.CheckpointID != "b20f84567474" {
		t.Fatalf("checkpoint = %q, want b20f84567474", res.CheckpointID)
	}
	if res.Note == "" {
		t.Fatal("expected a note explaining the fallback")
	}
}

func TestResolveCommitOnlyWhenNoCheckpoints(t *testing.T) {
	f := runner.NewFake()
	f.Set("entire", []string{"checkpoint", "list", "--json"}, runner.Result{Stdout: "[]"})
	f.Set("git", []string{"-C", "/repo", "rev-parse", "--verify", "HEAD^{commit}"}, runner.Result{Stdout: "abc1234\n"})
	f.Set("git", []string{"-C", "/repo", "rev-list", "--parents", "-n", "1", "abc1234"}, runner.Result{Stdout: "abc1234 def5678\n"})
	f.Set("git", []string{"-C", "/repo", "log", "-1", "--format=%B", "abc1234"}, runner.Result{Stdout: "no trailer here\n"})
	f.Set("git", []string{"-C", "/repo", "log", "-1", "--format=%cI", "abc1234"}, runner.Result{Stdout: "2026-09-04T17:58:00+00:00\n"})

	res, err := newResolver(f).Resolve(context.Background(), "HEAD")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if res.Route != RouteCommitOnly {
		t.Fatalf("route = %q, want %q", res.Route, RouteCommitOnly)
	}
	if res.CheckpointID != "" {
		t.Fatalf("checkpoint = %q, want empty", res.CheckpointID)
	}
}

func TestResolveMergeCommitUsesFirstParent(t *testing.T) {
	f := runner.NewFake()
	f.Set("entire", []string{"checkpoint", "list", "--json"}, runner.Result{Stdout: "[]"})
	f.Set("git", []string{"-C", "/repo", "rev-parse", "--verify", "HEAD^{commit}"}, runner.Result{Stdout: "merge01\n"})
	f.Set("git", []string{"-C", "/repo", "rev-list", "--parents", "-n", "1", "merge01"},
		runner.Result{Stdout: "merge01 first01 second02\n"})
	f.Set("git", []string{"-C", "/repo", "log", "-1", "--format=%B", "merge01"}, runner.Result{Stdout: "merge branch\n"})
	f.Set("git", []string{"-C", "/repo", "log", "-1", "--format=%cI", "merge01"}, runner.Result{Stdout: "2026-09-04T17:58:00+00:00\n"})

	res, err := newResolver(f).Resolve(context.Background(), "HEAD")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !res.IsMerge {
		t.Fatal("expected IsMerge")
	}
	if res.Parent != "first01" {
		t.Fatalf("parent = %q, want first01", res.Parent)
	}
	if res.Note == "" {
		t.Fatal("expected a note naming the first-parent choice")
	}
}

func TestResolveByCheckpointIDPrefix(t *testing.T) {
	f := runner.NewFake()
	f.Set("entire", []string{"checkpoint", "list", "--json"}, runner.Result{Stdout: probeListing})
	f.Set("git", []string{"-C", "/repo", "log", "--all", "--format=%H", "--grep", "Entire-Checkpoint: b20f84567474"},
		runner.Result{Stdout: "0063443\n"})
	f.Set("git", []string{"-C", "/repo", "rev-parse", "--verify", "0063443^{commit}"}, runner.Result{Stdout: "0063443\n"})
	f.Set("git", []string{"-C", "/repo", "rev-list", "--parents", "-n", "1", "0063443"}, runner.Result{Stdout: "0063443 faf4dc6\n"})

	res, err := newResolver(f).Resolve(context.Background(), "b20f84")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if res.Route != RouteID {
		t.Fatalf("route = %q, want %q", res.Route, RouteID)
	}
	if res.CheckpointID != "b20f84567474" {
		t.Fatalf("checkpoint = %q", res.CheckpointID)
	}
}

// ARCHITECTURE.md described the trailer as 12 hex characters. The installed
// CLI writes a 26 character ULID. Matching only hex silently missed every real
// trailer, so this pins the shapes that actually occur.
func TestParseTrailerAcceptsRealIDShapes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"ULID from the git hook", "phase 4\n\nEntire-Checkpoint: 01M1PTJGEYNGRB6R0H85Z29FKM\n", "01M1PTJGEYNGRB6R0H85Z29FKM"},
		{"twelve hex from import", "x\n\nEntire-Checkpoint: b20f84567474\n", "b20f84567474"},
		{"forty hex carry forward", "x\n\nEntire-Checkpoint: 041f789767de4d1180666bbf8346548f4481f996\n", "041f789767de4d1180666bbf8346548f4481f996"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ParseTrailer(c.body); got != c.want {
				t.Fatalf("ParseTrailer = %q, want %q", got, c.want)
			}
		})
	}
}

func TestLooksLikeCheckpointIDAcceptsULID(t *testing.T) {
	if !LooksLikeCheckpointID("01M1PTJGEYNGRB6R0H85Z29FKM") {
		t.Fatal("a ULID is a valid checkpoint id")
	}
	if !LooksLikeCheckpointID("b20f84567474") {
		t.Fatal("a hex id is a valid checkpoint id")
	}
	if LooksLikeCheckpointID("HEAD") {
		t.Fatal("HEAD is too short to be a checkpoint id")
	}
	if LooksLikeCheckpointID("feature/my-branch") {
		t.Fatal("a branch name is not a checkpoint id")
	}
}

// A ULID trailer must survive the whole resolve path, not just the regex.
func TestResolveByULIDTrailer(t *testing.T) {
	f := runner.NewFake()
	f.Set("entire", []string{"checkpoint", "list", "--json"}, runner.Result{Stdout: "[]"})
	f.Set("git", []string{"-C", "/repo", "rev-parse", "--verify", "HEAD^{commit}"}, runner.Result{Stdout: "ecb6388\n"})
	f.Set("git", []string{"-C", "/repo", "rev-list", "--parents", "-n", "1", "ecb6388"}, runner.Result{Stdout: "ecb6388 cfc38e2\n"})
	f.Set("git", []string{"-C", "/repo", "log", "-1", "--format=%B", "ecb6388"},
		runner.Result{Stdout: "umbra: phase 4\n\nEntire-Checkpoint: 01M1PTJGEYNGRB6R0H85Z29FKM\n"})

	res, err := newResolver(f).Resolve(context.Background(), "HEAD")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if res.Route != RouteTrailer {
		t.Fatalf("route = %q, want %q", res.Route, RouteTrailer)
	}
	if res.CheckpointID != "01M1PTJGEYNGRB6R0H85Z29FKM" {
		t.Fatalf("checkpoint = %q", res.CheckpointID)
	}
}
