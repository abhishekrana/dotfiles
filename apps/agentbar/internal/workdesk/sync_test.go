package workdesk

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeFetcher serves the fixture snapshot as if it came from GitLab, so the whole sync
// path runs with no forge: paging, decode, index build, document render and the atomic
// move into place.
type fakeFetcher struct {
	mrs, issues, todos []json.RawMessage
	todoErr            error
	calls              []string
}

func newFakeFetcher(t *testing.T) *fakeFetcher {
	t.Helper()
	f := &fakeFetcher{}
	m, err := FixtureMirror()
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	f.mrs = remarshal(t, m.MRs)
	f.issues = remarshal(t, m.Issues)
	f.todos = remarshal(t, m.Todos)
	return f
}

func (f *fakeFetcher) Host(context.Context) (string, error) { return "gitlab.example.com", nil }
func (f *fakeFetcher) User(context.Context) (string, error) { return "you", nil }

func (f *fakeFetcher) MergeRequests(_ context.Context, p, who string) ([]json.RawMessage, error) {
	f.calls = append(f.calls, "mrs:"+p+":"+who)
	return f.mrs, nil
}

func (f *fakeFetcher) Issues(_ context.Context, p, who string) ([]json.RawMessage, error) {
	f.calls = append(f.calls, "issues:"+p+":"+who)
	return f.issues, nil
}

func (f *fakeFetcher) Todos(_ context.Context, p string) ([]json.RawMessage, error) {
	f.calls = append(f.calls, "todos:"+p)
	return f.todos, f.todoErr
}

// restamp accounts for the one line a fresh sync legitimately changes.
//
// The golden documents carry the fixture's own hardcoded meta.synced, while their ages
// were computed against the clock that captured them - two different instants. A real
// sync stamps the moment it ran, so that line is substituted before diffing and
// everything else must match exactly.
func restamp(t *testing.T, dir, golden string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		t.Fatalf("read synced meta: %v", err)
	}
	var meta Meta
	if err := json.Unmarshal(b, &meta); err != nil {
		t.Fatalf("decode synced meta: %v", err)
	}
	return strings.ReplaceAll(golden, "2026-08-27 10:31:02", meta.Synced)
}

// A sync must produce a mirror that renders exactly what the golden files say, which
// makes this the end-to-end proof: fetch, decode, index, render, move into place, read
// back.
func TestSyncProducesAMirrorThatRendersTheGoldens(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "mirror")
	f := newFakeFetcher(t)
	now := frozen(t)

	res, err := Sync(context.Background(), f, dir, "acme/platform", now)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	// Derived from the fixture rather than hardcoded, so extending it to cover a new
	// case does not break an unrelated assertion.
	want, err := FixtureMirror()
	if err != nil {
		t.Fatal(err)
	}
	if res.MRs != len(want.MRs) || res.Issues != len(want.Issues) || res.Todos != len(want.Todos) {
		t.Errorf("synced %d mrs, %d issues, %d todos; want %d/%d/%d",
			res.MRs, res.Issues, res.Todos, len(want.MRs), len(want.Issues), len(want.Todos))
	}

	// The index on disk is what every interactive command reads, so it is the thing
	// worth checking rather than the snapshot it came from.
	idx, err := LoadIndex(dir)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	for _, c := range []struct {
		view   View
		golden string
	}{
		{ViewMRs, "list-mrs.golden"},
		{ViewIssues, "list-issues.golden"},
		{ViewInbox, "list-inbox.golden"},
	} {
		t.Run(c.view.String(), func(t *testing.T) {
			var got strings.Builder
			if err := WriteRows(&got, idx.Rows(c.view, now)); err != nil {
				t.Fatal(err)
			}
			diffLines(t, golden(t, c.golden), got.String())
		})
	}

	// Documents are pre-rendered at sync time so a preview is a file read.
	t.Run("sheet on disk", func(t *testing.T) {
		b, err := os.ReadFile(filepath.Join(dir, "mr", "412.md"))
		if err != nil {
			t.Fatalf("read rendered sheet: %v", err)
		}
		diffLines(t, restamp(t, dir, golden(t, filepath.Join("mr", "412.md"))), string(b))
	})
	t.Run("issue preview on disk", func(t *testing.T) {
		b, err := os.ReadFile(filepath.Join(dir, "issue", "128.md"))
		if err != nil {
			t.Fatalf("read rendered issue: %v", err)
		}
		diffLines(t, golden(t, "preview-issue-128.golden"), string(b))
	})
}

// The mirror is a full snapshot, and the point of that is deletion: work that merged or
// closed has to leave, not linger.
func TestSyncDropsWhatLeftTheQueue(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "mirror")
	f := newFakeFetcher(t)
	now := frozen(t)

	if _, err := Sync(context.Background(), f, dir, "acme/platform", now); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mr", "412.md")); err != nil {
		t.Fatalf("!412 should be in the first mirror: %v", err)
	}

	// !412 merges; the next snapshot simply does not contain it. Dropped by iid rather
	// than by position, so a reordered fixture cannot make this test pass vacuously.
	kept := f.mrs[:0:0]
	for _, raw := range f.mrs {
		var mr struct {
			IID string `json:"iid"`
		}
		if err := json.Unmarshal(raw, &mr); err != nil {
			t.Fatal(err)
		}
		if mr.IID != "412" {
			kept = append(kept, raw)
		}
	}
	if len(kept) != len(f.mrs)-1 {
		t.Fatalf("fixture no longer contains !412")
	}
	f.mrs = kept
	if _, err := Sync(context.Background(), f, dir, "acme/platform", now); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mr", "412.md")); !os.IsNotExist(err) {
		t.Error("!412 merged but its sheet survived the next sync")
	}
	idx, err := LoadIndex(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range idx.MRs {
		if it.Ref == "!412" {
			t.Error("!412 merged but is still in the index")
		}
	}
}

// A todo failure must not cost you the whole mirror: the feed needs a scope the token
// may not have, and every inferred band works without it.
func TestSyncSurvivesATodoFailure(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "mirror")
	f := newFakeFetcher(t)
	f.todoErr = errors.New("403 Forbidden")

	res, err := Sync(context.Background(), f, dir, "acme/platform", frozen(t))
	if err != nil {
		t.Fatalf("a todo failure should not fail the sync: %v", err)
	}
	if res.Todos != 0 {
		t.Errorf("got %d todos from a failing feed", res.Todos)
	}
	if res.MRs == 0 {
		t.Error("merge requests were lost along with the todos")
	}
}

// An index written before todos existed, or by an older version, must still open.
func TestLoadToleratesAnOlderMirror(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mrs.json"), []byte(`[]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err != nil {
		t.Errorf("a mirror with only mrs.json should load, got %v", err)
	}
}

func TestLoadWithoutAMirrorSaysSo(t *testing.T) {
	t.Parallel()
	if _, err := Load(t.TempDir()); !errors.Is(err, ErrNoMirror) {
		t.Errorf("Load on an empty dir = %v, want ErrNoMirror so the caller can say 'run sync'", err)
	}
	if _, err := LoadIndex(t.TempDir()); !errors.Is(err, ErrNoMirror) {
		t.Errorf("LoadIndex on an empty dir = %v, want ErrNoMirror", err)
	}
}

// Sync stages into a temp dir and moves it into place, so a failure must leave the
// previous mirror exactly as it was.
func TestFailedSyncLeavesThePreviousMirrorIntact(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "mirror")
	f := newFakeFetcher(t)
	now := frozen(t)
	if _, err := Sync(context.Background(), f, dir, "acme/platform", now); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	before, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}

	broken := &failingFetcher{fakeFetcher: f}
	if _, err := Sync(context.Background(), broken, dir, "acme/platform", now); err == nil {
		t.Fatal("a failing fetch reported success")
	}
	after, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatalf("the previous index is gone: %v", err)
	}
	if string(before) != string(after) {
		t.Error("a failed sync modified the previous mirror")
	}
}

type failingFetcher struct{ *fakeFetcher }

func (f *failingFetcher) MergeRequests(context.Context, string, string) ([]json.RawMessage, error) {
	return nil, errors.New("the forge went away")
}

func TestRenderRebuildsWithoutANetwork(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "mirror")
	f := newFakeFetcher(t)
	now := frozen(t)
	if _, err := Sync(context.Background(), f, dir, "acme/platform", now); err != nil {
		t.Fatal(err)
	}
	// Throw the derived tiers away; Render must put them back from the snapshot alone.
	for _, p := range []string{"index.json", "board.md"} {
		if err := os.Remove(filepath.Join(dir, p)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Render(dir, now); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if _, err := LoadIndex(dir); err != nil {
		t.Errorf("Render did not rebuild the index: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "board.md"))
	if err != nil {
		t.Fatalf("Render did not rebuild the board: %v", err)
	}
	want := strings.Replace(golden(t, "board.golden"), "`work mr <iid>`", "`workdesk mr <iid>`", 1)
	diffLines(t, restamp(t, dir, want), string(b))
}

// remarshal turns decoded fixture values back into the raw nodes a fetcher hands over,
// so the fake exercises the same decode path the real one does.
func remarshal[T any](t *testing.T, items []T) []json.RawMessage {
	t.Helper()
	out := make([]json.RawMessage, 0, len(items))
	for _, it := range items {
		b, err := json.Marshal(it)
		if err != nil {
			t.Fatalf("remarshal: %v", err)
		}
		out = append(out, b)
	}
	return out
}
