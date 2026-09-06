package checkpoint

import "testing"

// Real output captured from `entire checkpoint explain <id> --short` in the
// Step 0 probe. Imported checkpoints carry no summary.
const probeShortNoSummary = `● Checkpoint b20f84567474
  session  8017734d-cdf6-4a18-b2f8-f4b5f637ec35
  created  2026-09-04 17:54:32
  author   A Person <someone@example.com>
  tokens   4857.9k
  commits  (none on this branch)
────────────────────────────────────────────────────────────
## Intent

then make a repo named Umbra on my github, ignore kickoff and handoff, kickof...

## Summary

*No summary. Imported history is read-only, so summaries cannot be generated.*
`

const probeShortWithSummary = `● Checkpoint a1b2c3d4e5f6
  session  1111
────────────────────────────────────────────────────────────
## Intent

change the totals

## Summary

Checked the callers of compute_total; all tests pass.
`

func TestParseSummaryAbsentOnImportedCheckpoint(t *testing.T) {
	if got := ParseSummary(probeShortNoSummary); got != "" {
		t.Fatalf("ParseSummary = %q, want empty so the caller falls back", got)
	}
}

func TestParseSummaryPresent(t *testing.T) {
	want := "Checked the callers of compute_total; all tests pass."
	if got := ParseSummary(probeShortWithSummary); got != want {
		t.Fatalf("ParseSummary = %q, want %q", got, want)
	}
}

func TestParseSummaryMissingHeading(t *testing.T) {
	if got := ParseSummary("nothing useful here\n"); got != "" {
		t.Fatalf("ParseSummary = %q, want empty", got)
	}
}

func TestIntent(t *testing.T) {
	want := "then make a repo named Umbra on my github, ignore kickoff and handoff, kickof..."
	if got := Intent(probeShortNoSummary); got != want {
		t.Fatalf("Intent = %q, want %q", got, want)
	}
}

// A checkpoint written by the git hook has no summary either, but Entire
// words the placeholder differently from the imported case.
const probeShortNotGenerated = `● Checkpoint 01M1PVD1WBK1J1J8GWDNJJR7XZ
  session  8017734d
────────────────────────────────────────────────────────────
## Intent

require an explicit currency

## Summary

*Not generated yet. Run 'entire checkpoint explain --generate 01M1PVD1WBK1J1J8GWDNJJR7XZ' to create an AI summary.*
`

func TestParseSummaryAbsentWhenNotGenerated(t *testing.T) {
	if got := ParseSummary(probeShortNotGenerated); got != "" {
		t.Fatalf("ParseSummary = %q, want empty so the caller falls back", got)
	}
}

func TestParseSummaryKeepsRealProse(t *testing.T) {
	text := "## Summary\n\nChecked the callers. All tests pass.\n"
	if got := ParseSummary(text); got != "Checked the callers. All tests pass." {
		t.Fatalf("ParseSummary = %q", got)
	}
}
