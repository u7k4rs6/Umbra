// Package checkpoint turns whatever the user typed into a checkpoint id, a
// commit and its parent, and fetches the transcript for that checkpoint.
// Every external process goes through the Runner it is given.
package checkpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

// Route names how a reference was resolved. It is printed in the header so a
// reader knows how much to trust the pairing.
const (
	RouteTrailer       = "trailer"        // the commit carries an Entire-Checkpoint trailer
	RouteID            = "checkpoint id"  // the id was given and its commit was found
	RouteSessionWindow = "session window" // no trailer, paired by session time
	RouteCommitOnly    = "commit only"    // no checkpoint could be paired at all
)

// Resolution is the answer: which checkpoint, which commit, which parent.
type Resolution struct {
	CheckpointID string
	Commit       string
	Parent       string
	IsMerge      bool
	Route        string
	Agent        string
	SessionIDs   []string
	// Note explains a fallback in one sentence for the report header.
	Note string
}

// Listing is one entry of `entire checkpoint list --json`.
type Listing struct {
	CheckpointID string   `json:"checkpoint_id"`
	SessionID    string   `json:"session_id"`
	Agent        string   `json:"agent"`
	Date         string   `json:"date"`
	Message      string   `json:"message"`
	IsLogsOnly   bool     `json:"is_logs_only"`
	SessionCount int      `json:"session_count"`
	SessionIDs   []string `json:"session_ids"`
}

// Time parses the listing date, returning the zero time when it is unusable.
func (l Listing) Time() time.Time {
	t, err := time.Parse(time.RFC3339, l.Date)
	if err != nil {
		return time.Time{}
	}
	return t
}

// Checkpoint ids are not hexadecimal. ARCHITECTURE.md described a 12 hex
// trailer, but the installed CLI writes a 26 character ULID, for example
// "Entire-Checkpoint: 01M1PTJGEYNGRB6R0H85Z29FKM", alongside 12 hex ids for
// imported history and 40 hex ids for carry-forward entries. Matching only hex
// silently missed every real trailer, so the pattern accepts any alphanumeric
// id and the length range covers all three shapes.
var (
	trailerRE = regexp.MustCompile(`(?m)^Entire-Checkpoint:\s*([0-9A-Za-z]{6,40})\s*$`)
	idRE      = regexp.MustCompile(`^[0-9A-Za-z]{6,40}$`)
)

// ParseTrailer returns the checkpoint id carried by a commit message, or the
// empty string when there is none.
func ParseTrailer(message string) string {
	m := trailerRE.FindStringSubmatch(message)
	if m == nil {
		return ""
	}
	// Ids are compared case insensitively but returned as written, because a
	// ULID is uppercase and git log must be grepped for the exact text.
	return m[1]
}

// LooksLikeCheckpointID reports whether ref has the shape of a checkpoint id.
// A short hex string is ambiguous with an abbreviated commit sha, so the
// resolver tries both and prefers whichever git and Entire agree on.
func LooksLikeCheckpointID(ref string) bool { return idRE.MatchString(ref) }

// Resolver holds the dependencies resolution needs.
type Resolver struct {
	Run  runner.Runner
	Repo string
	// Now is injected so the session-window fallback is testable.
	Now func() time.Time
}

// New builds a resolver rooted at repo.
func New(run runner.Runner, repo string) *Resolver {
	return &Resolver{Run: run, Repo: repo, Now: time.Now}
}

// Resolve turns ref into a full Resolution.
//
// Order: an Entire-Checkpoint trailer on the commit is the truth when it is
// there. Failing that, a reference that parses as a checkpoint id is looked up
// in the checkpoint list and matched to a commit by trailer search. Failing
// that, the reference is treated as a commit-ish and paired with the
// checkpoint whose session window contains the commit time. Failing even that,
// the commit stands alone and every node is unknown, which is the correct
// answer rather than an error.
func (r *Resolver) Resolve(ctx context.Context, ref string) (*Resolution, error) {
	if strings.TrimSpace(ref) == "" {
		ref = "HEAD"
	}

	listings, listErr := r.List(ctx)

	// A reference that names a known checkpoint takes that path first.
	if LooksLikeCheckpointID(ref) {
		if l, ok := matchListing(listings, ref); ok {
			res := &Resolution{
				CheckpointID: l.CheckpointID,
				Route:        RouteID,
				Agent:        l.Agent,
				SessionIDs:   l.SessionIDs,
			}
			sha, err := r.commitForCheckpoint(ctx, l.CheckpointID)
			if err == nil && sha != "" {
				if err := r.fillCommit(ctx, res, sha); err != nil {
					return nil, err
				}
				return res, nil
			}
			// The checkpoint exists but no commit carries its trailer. This is
			// the imported-history case the probe found.
			sha, err = r.revParse(ctx, "HEAD")
			if err != nil {
				return nil, fmt.Errorf("checkpoint %s has no commit and HEAD is unreadable: %w", l.CheckpointID, err)
			}
			if err := r.fillCommit(ctx, res, sha); err != nil {
				return nil, err
			}
			res.Route = RouteSessionWindow
			res.Note = fmt.Sprintf("checkpoint %s carries no commit, so the change was read from HEAD", short(l.CheckpointID))
			return res, nil
		}
	}

	// Treat it as a commit-ish.
	sha, err := r.revParse(ctx, ref)
	if err != nil {
		if listErr != nil {
			return nil, fmt.Errorf("%q is neither a checkpoint nor a commit: %v", ref, err)
		}
		return nil, fmt.Errorf("%q is neither a checkpoint nor a commit: %v", ref, err)
	}

	res := &Resolution{}
	if err := r.fillCommit(ctx, res, sha); err != nil {
		return nil, err
	}

	body, err := r.commitBody(ctx, sha)
	if err != nil {
		return nil, err
	}
	if id := ParseTrailer(body); id != "" {
		res.CheckpointID = id
		res.Route = RouteTrailer
		if l, ok := matchListing(listings, id); ok {
			res.Agent = l.Agent
			res.SessionIDs = l.SessionIDs
		}
		return res, nil
	}

	// No trailer. Pair with the checkpoint whose session was open across the
	// commit, which is what happens when hooks were not loaded for the session.
	when, err := r.commitTime(ctx, sha)
	if err == nil {
		if l, ok := nearestListing(listings, when); ok {
			res.CheckpointID = l.CheckpointID
			res.Route = RouteSessionWindow
			res.Agent = l.Agent
			res.SessionIDs = l.SessionIDs
			res.Note = fmt.Sprintf(
				"commit %s carries no Entire-Checkpoint trailer, so it was paired with checkpoint %s by session time",
				short(sha), short(l.CheckpointID))
			return res, nil
		}
	}

	res.Route = RouteCommitOnly
	res.Note = fmt.Sprintf("no checkpoint could be paired with commit %s, so the examined set is unavailable", short(sha))
	return res, nil
}

// List returns the checkpoints Entire knows about locally.
func (r *Resolver) List(ctx context.Context) ([]Listing, error) {
	stdout, _, exit, err := r.Run.Run(ctx, "entire", []string{"checkpoint", "list", "--json"}, nil)
	if err != nil {
		return nil, err
	}
	if exit != 0 {
		return nil, fmt.Errorf("entire checkpoint list exited %d", exit)
	}
	return ParseListing(stdout)
}

// ParseListing reads the JSON array `entire checkpoint list --json` prints.
//
// Entire prefixes a warning line when the checkpoint remote is unreachable,
// and that warning itself begins with a bracket ("[entire] Warning: ..."), so
// the parser cannot simply seek to the first bracket. It tries each bracket in
// turn and keeps the first one that parses as an array of listings.
func ParseListing(stdout []byte) ([]Listing, error) {
	text := string(stdout)
	var lastErr error
	for i, r := range text {
		if r != '[' {
			continue
		}
		var out []Listing
		if err := json.Unmarshal([]byte(text[i:]), &out); err != nil {
			lastErr = err
			continue
		}
		return out, nil
	}
	if lastErr == nil {
		return nil, nil
	}
	return nil, fmt.Errorf("checkpoint list: %w", lastErr)
}

func (r *Resolver) fillCommit(ctx context.Context, res *Resolution, sha string) error {
	res.Commit = sha
	parents, err := r.parents(ctx, sha)
	if err != nil {
		return err
	}
	switch len(parents) {
	case 0:
		res.Parent = ""
	default:
		res.Parent = parents[0]
		res.IsMerge = len(parents) > 1
		if res.IsMerge && res.Note == "" {
			res.Note = fmt.Sprintf("commit %s is a merge, so the first parent was used as the base", short(sha))
		}
	}
	return nil
}

func (r *Resolver) commitForCheckpoint(ctx context.Context, id string) (string, error) {
	args := []string{"-C", r.Repo, "log", "--all", "--format=%H", "--grep", "Entire-Checkpoint: " + id}
	stdout, _, exit, err := r.Run.Run(ctx, "git", args, nil)
	if err != nil || exit != 0 {
		return "", fmt.Errorf("git log for checkpoint %s failed", id)
	}
	lines := nonEmptyLines(string(stdout))
	if len(lines) == 0 {
		return "", nil
	}
	return lines[0], nil
}

func (r *Resolver) revParse(ctx context.Context, ref string) (string, error) {
	stdout, stderr, exit, err := r.Run.Run(ctx, "git", []string{"-C", r.Repo, "rev-parse", "--verify", ref + "^{commit}"}, nil)
	if err != nil {
		return "", err
	}
	if exit != 0 {
		return "", &runner.ExitError{Call: runner.Call{Name: "git", Args: []string{"rev-parse", ref}}, Exit: exit, Stderr: string(stderr)}
	}
	out := strings.TrimSpace(string(stdout))
	if out == "" {
		return "", fmt.Errorf("git rev-parse %s produced nothing", ref)
	}
	return out, nil
}

func (r *Resolver) parents(ctx context.Context, sha string) ([]string, error) {
	stdout, _, exit, err := r.Run.Run(ctx, "git", []string{"-C", r.Repo, "rev-list", "--parents", "-n", "1", sha}, nil)
	if err != nil {
		return nil, err
	}
	if exit != 0 {
		return nil, fmt.Errorf("git rev-list for %s exited %d", short(sha), exit)
	}
	fields := strings.Fields(strings.TrimSpace(string(stdout)))
	if len(fields) <= 1 {
		return nil, nil
	}
	return fields[1:], nil
}

func (r *Resolver) commitBody(ctx context.Context, sha string) (string, error) {
	stdout, _, exit, err := r.Run.Run(ctx, "git", []string{"-C", r.Repo, "log", "-1", "--format=%B", sha}, nil)
	if err != nil {
		return "", err
	}
	if exit != 0 {
		return "", fmt.Errorf("git log for %s exited %d", short(sha), exit)
	}
	return string(stdout), nil
}

func (r *Resolver) commitTime(ctx context.Context, sha string) (time.Time, error) {
	stdout, _, exit, err := r.Run.Run(ctx, "git", []string{"-C", r.Repo, "log", "-1", "--format=%cI", sha}, nil)
	if err != nil {
		return time.Time{}, err
	}
	if exit != 0 {
		return time.Time{}, fmt.Errorf("git log for %s exited %d", short(sha), exit)
	}
	return time.Parse(time.RFC3339, strings.TrimSpace(string(stdout)))
}

// matchListing finds a checkpoint by exact id or unique prefix.
func matchListing(listings []Listing, ref string) (Listing, bool) {
	lower := strings.ToLower(ref)
	for _, l := range listings {
		if strings.EqualFold(l.CheckpointID, ref) {
			return l, true
		}
	}
	var hits []Listing
	for _, l := range listings {
		if strings.HasPrefix(strings.ToLower(l.CheckpointID), lower) {
			hits = append(hits, l)
		}
	}
	if len(hits) == 1 {
		return hits[0], true
	}
	return Listing{}, false
}

// nearestListing picks the checkpoint closest in time to when, preferring one
// recorded at or before the commit because a session precedes the commit it
// produced. Ties break on the id so the choice is deterministic.
func nearestListing(listings []Listing, when time.Time) (Listing, bool) {
	type scored struct {
		l    Listing
		dist time.Duration
		past bool
	}
	var cands []scored
	for _, l := range listings {
		t := l.Time()
		if t.IsZero() {
			continue
		}
		d := when.Sub(t)
		past := d >= 0
		if d < 0 {
			d = -d
		}
		cands = append(cands, scored{l: l, dist: d, past: past})
	}
	if len(cands) == 0 {
		return Listing{}, false
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].past != cands[j].past {
			return cands[i].past
		}
		if cands[i].dist != cands[j].dist {
			return cands[i].dist < cands[j].dist
		}
		return cands[i].l.CheckpointID < cands[j].l.CheckpointID
	})
	return cands[0].l, true
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, ln := range strings.Split(s, "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			out = append(out, ln)
		}
	}
	return out
}

func short(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}
