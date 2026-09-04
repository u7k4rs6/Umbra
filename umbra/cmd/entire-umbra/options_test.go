package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitWords(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"plain", "pytest -v", []string{"pytest", "-v"}},
		{"extra spaces", "  pytest   -v  ", []string{"pytest", "-v"}},
		{"single quotes", "go test -run 'TestA|TestB' ./...", []string{"go", "test", "-run", "TestA|TestB", "./..."}},
		{"double quotes", `pytest -k "not slow"`, []string{"pytest", "-k", "not slow"}},
		{"escaped space", `my\ runner -v`, []string{"my runner", "-v"}},
		{"empty", "", nil},
		{"quoted empty argument", `pytest ""`, []string{"pytest", ""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := SplitWords(c.in)
			if err != nil {
				t.Fatalf("SplitWords(%q): %v", c.in, err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("SplitWords(%q) = %#v, want %#v", c.in, got, c.want)
			}
		})
	}
}

func TestSplitWordsUnbalancedQuote(t *testing.T) {
	if _, err := SplitWords(`pytest -k "not slow`); err == nil {
		t.Fatal("expected an error for an unbalanced quote")
	}
}

// Umbra never builds a shell string, so a runner prefix carrying shell syntax
// stays inert: it becomes ordinary argv elements.
func TestSplitWordsDoesNotInterpretShellSyntax(t *testing.T) {
	got, err := SplitWords("pytest; rm -rf /")
	if err != nil {
		t.Fatalf("SplitWords: %v", err)
	}
	want := []string{"pytest;", "rm", "-rf", "/"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func base() *Options {
	return &Options{Ref: "HEAD", Run: "none", Depth: 2, Format: "table", Adapter: "auto"}
}

func TestValidateAcceptsDefaults(t *testing.T) {
	if err := base().Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := []struct {
		name  string
		mut   func(*Options)
		match string
	}{
		{"bad run", func(o *Options) { o.Run = "some" }, "--run"},
		{"bad format", func(o *Options) { o.Format = "pdf" }, "--format"},
		{"bad fail-on", func(o *Options) { o.FailOn = "everything" }, "--fail-on"},
		{"bad adapter", func(o *Options) { o.Adapter = "cursor" }, "--adapter"},
		{"depth too low", func(o *Options) { o.Depth = 0 }, "--depth"},
		{"depth too high", func(o *Options) { o.Depth = 4 }, "--depth"},
		{"run without test", func(o *Options) { o.Run = "shadow" }, "test runner"},
		{"html without out", func(o *Options) { o.Format = "html" }, "--out"},
		{"packet without out", func(o *Options) { o.Format = "packet" }, "--out"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := base()
			c.mut(o)
			err := o.Validate()
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), c.match) {
				t.Fatalf("error %q does not mention %q", err, c.match)
			}
		})
	}
}

func TestValidateRunNoneNeedsNoTestRunner(t *testing.T) {
	o := base()
	o.Run = "none"
	o.Test = ""
	if err := o.Validate(); err != nil {
		t.Fatalf("--run none must not require --test: %v", err)
	}
}

func TestParseFlagsDefaults(t *testing.T) {
	t.Setenv("ENTIRE_REPO_ROOT", t.TempDir())
	o, err := parseFlags([]string{"--run", "none"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if o.Ref != "HEAD" {
		t.Fatalf("ref = %q, want HEAD", o.Ref)
	}
	if o.Depth != 2 {
		t.Fatalf("depth = %d, want 2", o.Depth)
	}
	if o.Format != "table" {
		t.Fatalf("format = %q", o.Format)
	}
	if o.NoAudit {
		t.Fatal("the sweep is on by default, so --no-audit must default to false")
	}
	if o.History {
		t.Fatal("the scar factor is opt in")
	}
	if o.Snippets {
		t.Fatal("snippets are off by default")
	}
}

func TestParseFlagsTakesRefPositionally(t *testing.T) {
	t.Setenv("ENTIRE_REPO_ROOT", t.TempDir())
	o, err := parseFlags([]string{"--run", "none", "b20f84567474"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if o.Ref != "b20f84567474" {
		t.Fatalf("ref = %q", o.Ref)
	}
}

func TestParseFlagsUsesRepoRootFromEnvironment(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ENTIRE_REPO_ROOT", dir)
	o, err := parseFlags([]string{"--run", "none"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if o.Repo != dir {
		t.Fatalf("repo = %q, want %q", o.Repo, dir)
	}
}

// PRD.md and the demo script say "pytest -q", but the Step 0 probe showed
// verify cannot parse quiet output. The default carries the correction.
func TestDefaultTestRunnerIsVerbose(t *testing.T) {
	if !strings.Contains(DefaultTestRunner, "-v") {
		t.Fatalf("DefaultTestRunner = %q, want the verbose form verify can parse", DefaultTestRunner)
	}
}

// The PRD's command surface is `entire umbra <ref> [flags]`, so flags typed
// after the reference must still take effect. The standard flag package stops
// at the first positional, which would drop them silently.
func TestParseFlagsAcceptsFlagsAfterTheReference(t *testing.T) {
	t.Setenv("ENTIRE_REPO_ROOT", t.TempDir())
	o, err := parseFlags([]string{"HEAD", "--run", "none", "--no-audit", "--depth", "3"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if o.Ref != "HEAD" {
		t.Fatalf("ref = %q", o.Ref)
	}
	if o.Run != "none" {
		t.Fatalf("run = %q, want none", o.Run)
	}
	if !o.NoAudit {
		t.Fatal("--no-audit after the reference was dropped")
	}
	if o.Depth != 3 {
		t.Fatalf("depth = %d, want 3", o.Depth)
	}
}

func TestParseFlagsBooleanDoesNotSwallowTheReference(t *testing.T) {
	t.Setenv("ENTIRE_REPO_ROOT", t.TempDir())
	o, err := parseFlags([]string{"--no-audit", "b20f84567474", "--run", "none"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if o.Ref != "b20f84567474" {
		t.Fatalf("ref = %q, want the checkpoint id", o.Ref)
	}
	if !o.NoAudit {
		t.Fatal("--no-audit was dropped")
	}
}

func TestParseFlagsValueFlagKeepsItsValue(t *testing.T) {
	t.Setenv("ENTIRE_REPO_ROOT", t.TempDir())
	o, err := parseFlags([]string{"--test", "pytest -v", "HEAD"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if o.Test != "pytest -v" {
		t.Fatalf("test = %q", o.Test)
	}
	if o.Ref != "HEAD" {
		t.Fatalf("ref = %q", o.Ref)
	}
}
