package workdesk

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// fakeFetcher serves the fixture snapshot as if it came from GitLab, so the whole sync
// path runs with no forge: paging, decode, index build, document render and the atomic
// move into place.
type fakeFetcher struct {
	mu                 sync.Mutex
	mrs, issues, todos []json.RawMessage
	todoErr            error
	calls              []string
	// fetched records the iids asked for in full, which is the whole point of the
	// manifest: a row GitLab says has not changed must not be downloaded again.
	fetched []string
}

func (f *fakeFetcher) note(call string, iids ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call)
	f.fetched = append(f.fetched, iids...)
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

// The manifest is asked once per account per relation and the answers unioned, so the
// fake answers the whole collection every time: the sync has to be the thing that drops
// the duplicates.
func (f *fakeFetcher) MergeRequestStamps(_ context.Context, p string, users []string) ([]json.RawMessage, error) {
	f.note("mr-stamps:" + p + ":" + strings.Join(users, ","))
	return stampsOf(f.mrs), nil
}

func (f *fakeFetcher) IssueStamps(_ context.Context, p string, users []string) ([]json.RawMessage, error) {
	f.note("issue-stamps:" + p + ":" + strings.Join(users, ","))
	return stampsOf(f.issues), nil
}

func (f *fakeFetcher) Statuses(_ context.Context, p string) ([]json.RawMessage, error) {
	f.note("statuses:" + p)
	m, err := FixtureMirror()
	if err != nil {
		return nil, err
	}
	out := make([]json.RawMessage, 0, len(m.Meta.Statuses))
	for _, st := range m.Meta.Statuses {
		b, err := json.Marshal(st)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func (f *fakeFetcher) CurrentIteration(_ context.Context, p string) (json.RawMessage, error) {
	f.note("iteration:" + p)
	m, err := FixtureMirror()
	if err != nil {
		return nil, err
	}
	if m.Meta.Iteration == nil {
		return nil, nil
	}
	return json.Marshal(m.Meta.Iteration)
}

func (f *fakeFetcher) MergeRequestsByIID(_ context.Context, p string, iids []string) ([]json.RawMessage, error) {
	f.note("mrs:"+p, iids...)
	return pickByIID(f.mrs, iids), nil
}

func (f *fakeFetcher) IssuesByIID(_ context.Context, p string, iids []string) ([]json.RawMessage, error) {
	f.note("issues:"+p, iids...)
	return pickByIID(f.issues, iids), nil
}

func (f *fakeFetcher) Todos(_ context.Context, p string, actions []string) ([]json.RawMessage, error) {
	f.note("todos:" + p + ":" + strings.Join(actions, ","))
	return f.todos, f.todoErr
}

// stampsOf is the manifest GitLab would answer with: identity and change token, nothing
// else.
func stampsOf(nodes []json.RawMessage) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(nodes))
	for _, n := range nodes {
		var st struct {
			IID       string `json:"iid"`
			UpdatedAt string `json:"updatedAt"`
		}
		if err := json.Unmarshal(n, &st); err != nil {
			continue
		}
		b, err := json.Marshal(st)
		if err != nil {
			continue
		}
		out = append(out, b)
	}
	return out
}

func pickByIID(nodes []json.RawMessage, iids []string) []json.RawMessage {
	want := make(map[string]bool, len(iids))
	for _, iid := range iids {
		want[iid] = true
	}
	var out []json.RawMessage
	for _, n := range nodes {
		var st struct {
			IID string `json:"iid"`
		}
		if err := json.Unmarshal(n, &st); err == nil && want[st.IID] {
			out = append(out, n)
		}
	}
	return out
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

	res, err := Sync(context.Background(), f, dir, "acme/platform", nil, now)
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

// Two accounts, and a row either of them owns is one row.
//
// The manifest is asked once per account per relation - author and assignee - because
// GitLab's own or: filter answered with a fraction of what its parts return. The fake
// answers the whole collection every time, which is the worst case: every row named four
// times over.
func TestSyncUnionsTheAccountsAndDeduplicates(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "mirror")
	f := newFakeFetcher(t)
	cfg := &Config{Accounts: []string{Self, "colleague"}}
	res, err := Sync(context.Background(), f, dir, "acme/platform", cfg, frozen(t))
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if got := strings.Join(res.Users, ","); got != "you,colleague" {
		t.Errorf("synced for %q, want \"you,colleague\"", got)
	}
	if got, want := len(res.Users), 2; got != want {
		t.Fatalf("%d accounts, want %d", got, want)
	}
	// One manifest call per account, and the caller sees the accounts it asked for.
	for _, want := range []string{"mr-stamps:acme/platform:you,colleague", "issue-stamps:acme/platform:you,colleague"} {
		if !slices.Contains(f.calls, want) {
			t.Errorf("no %q among %v", want, f.calls)
		}
	}
	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	seen := map[string]bool{}
	for _, is := range m.Issues {
		if seen[is.IID] {
			t.Errorf("#%s is in the mirror twice", is.IID)
		}
		seen[is.IID] = true
	}
	for _, mr := range m.MRs {
		if seen[mr.IID] {
			t.Errorf("!%s is in the mirror twice", mr.IID)
		}
		seen[mr.IID] = true
	}
	// A row named twice must still be fetched in full once.
	fetched := map[string]int{}
	for _, iid := range f.fetched {
		fetched[iid]++
	}
	for iid, n := range fetched {
		if n > 1 {
			t.Errorf("%s was downloaded %d times", iid, n)
		}
	}
}

// A mirror written by an older selection is refetched in full, once.
//
// The manifest only names rows GitLab says have changed, so a field added to the query
// would otherwise reach only the rows that happened to move: adding status left every
// quiet issue banded as "no status" with nothing to say why.
func TestSyncRefetchesWhenTheSelectionChanged(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "mirror")
	f := newFakeFetcher(t)
	now := frozen(t)
	first, err := Sync(context.Background(), f, dir, "acme/platform", nil, now)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if first.IssuesFetched == 0 {
		t.Fatal("the first sync fetched nothing")
	}

	// Nothing moved, so nothing is downloaded again.
	f.fetched = nil
	again, err := Sync(context.Background(), f, dir, "acme/platform", nil, now)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if again.IssuesFetched+again.MRsFetched != 0 {
		t.Errorf("an unchanged queue refetched %d rows", again.IssuesFetched+again.MRsFetched)
	}

	// Now the rows on disk are the wrong shape, and GitLab still says they have not
	// changed.
	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	m.Meta.Schema = MirrorSchema - 1
	if err := WriteMirror(dir, m, now); err != nil {
		t.Fatalf("WriteMirror: %v", err)
	}
	after, err := Sync(context.Background(), f, dir, "acme/platform", nil, now)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if after.IssuesFetched != after.Issues || after.MRsFetched != after.MRs {
		t.Errorf("refetched %d of %d issues and %d of %d merge requests, want all of both",
			after.IssuesFetched, after.Issues, after.MRsFetched, after.MRs)
	}
}

// The lifecycle and the sprint reach the mirror, because the issue bands and the sprint
// marker are read back out of it by every view.
func TestSyncRecordsTheWorkflow(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "mirror")
	f := newFakeFetcher(t)
	if _, err := Sync(context.Background(), f, dir, "acme/platform", nil, frozen(t)); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(m.Meta.Statuses) == 0 {
		t.Fatal("the mirror holds no lifecycle, so every issue would band as one")
	}
	if m.Meta.Statuses[0].Name != "Backlog" {
		t.Errorf("lifecycle starts at %q, want GitLab's own first column", m.Meta.Statuses[0].Name)
	}
	if m.Meta.Iteration == nil {
		t.Fatal("the mirror holds no current sprint, so no row could be marked")
	}
}

// The mirror is a full snapshot, and the point of that is deletion: work that merged or
// closed has to leave, not linger.
func TestSyncDropsWhatLeftTheQueue(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "mirror")
	f := newFakeFetcher(t)
	now := frozen(t)

	if _, err := Sync(context.Background(), f, dir, "acme/platform", nil, now); err != nil {
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
	if _, err := Sync(context.Background(), f, dir, "acme/platform", nil, now); err != nil {
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

	res, err := Sync(context.Background(), f, dir, "acme/platform", nil, frozen(t))
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
	if _, err := Sync(context.Background(), f, dir, "acme/platform", nil, now); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	before, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}

	broken := &failingFetcher{fakeFetcher: f}
	if _, err := Sync(context.Background(), broken, dir, "acme/platform", nil, now); err == nil {
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

func (f *failingFetcher) MergeRequestStamps(context.Context, string, []string) ([]json.RawMessage, error) {
	return nil, errors.New("the forge went away")
}

func TestRenderRebuildsWithoutANetwork(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "mirror")
	f := newFakeFetcher(t)
	now := frozen(t)
	if _, err := Sync(context.Background(), f, dir, "acme/platform", nil, now); err != nil {
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

// A sync is half a minute of network with the UI torn down, so the caller draws a line
// from these reports. Each leg must open and close exactly once, and close with the
// number of rows it brought back - a leg that never closes reads as stuck.
func TestSyncReportsEveryLeg(t *testing.T) {
	dir := t.TempDir()
	f := newFakeFetcher(t)
	now := frozen(t)

	var mu sync.Mutex
	opened := map[string]int{}
	closed := map[string]int{}
	counts := map[string]int{}
	report := func(leg string, done bool, n int) {
		mu.Lock()
		defer mu.Unlock()
		if done {
			closed[leg]++
			counts[leg] = n
			return
		}
		opened[leg]++
	}

	res, err := SyncWithProgress(context.Background(), f, dir, "acme/platform", nil, now, report)
	if err != nil {
		t.Fatal(err)
	}
	for _, leg := range []string{"identity", "merge requests", "issues", "todos", "writing"} {
		if opened[leg] != 1 || closed[leg] != 1 {
			t.Errorf("%s: opened %d, closed %d, want 1 and 1", leg, opened[leg], closed[leg])
		}
	}
	if counts["merge requests"] != res.MRs || counts["issues"] != res.Issues {
		t.Errorf("counts %d/%d do not match the result %d/%d",
			counts["merge requests"], counts["issues"], res.MRs, res.Issues)
	}
}

// The manifest is the whole optimisation: GitLab's updatedAt says which rows moved, and
// only those are downloaded. A detail node costs about 0.4s of the forge's time, so a
// sync that re-fetched an unchanged queue is the difference between one second and
// half a minute.
func TestSyncFetchesOnlyWhatChanged(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "mirror")
	f := newFakeFetcher(t)
	now := frozen(t)

	first, err := Sync(context.Background(), f, dir, "acme/platform", nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if first.MRsFetched != first.MRs || first.MRs == 0 {
		t.Fatalf("a first sync must fetch every row: got %d of %d", first.MRsFetched, first.MRs)
	}

	f.fetched = nil
	again, err := Sync(context.Background(), f, dir, "acme/platform", nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if again.MRsFetched != 0 || again.IssuesFetched != 0 {
		t.Errorf("nothing changed, yet %d merge requests and %d issues were fetched",
			again.MRsFetched, again.IssuesFetched)
	}
	if again.MRs != first.MRs || again.Issues != first.Issues {
		t.Errorf("the mirror lost rows it did not refetch: %d/%d, want %d/%d",
			again.MRs, again.Issues, first.MRs, first.Issues)
	}

	moved := bumpUpdatedAt(t, f.mrs[0])
	f.mrs[0] = moved.node
	f.fetched = nil
	third, err := Sync(context.Background(), f, dir, "acme/platform", nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if third.MRsFetched != 1 {
		t.Errorf("one merge request moved, %d were fetched", third.MRsFetched)
	}
	if len(f.fetched) != 1 || f.fetched[0] != moved.iid {
		t.Errorf("fetched %v, want just %s", f.fetched, moved.iid)
	}
}

// The manifest is also what keeps the snapshot full. A merge request that merged is
// simply absent from it, and must leave the mirror with it - the old sync got this for
// free by overwriting everything, and this one has to mean it.
func TestSyncDropsWhatLeftTheManifest(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "mirror")
	f := newFakeFetcher(t)
	now := frozen(t)

	first, err := Sync(context.Background(), f, dir, "acme/platform", nil, now)
	if err != nil {
		t.Fatal(err)
	}
	gone := iidOf(t, f.mrs[0])
	f.mrs = f.mrs[1:] // merged since the last sync

	after, err := Sync(context.Background(), f, dir, "acme/platform", nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if after.MRs != first.MRs-1 {
		t.Fatalf("want %d merge requests after one merged, got %d", first.MRs-1, after.MRs)
	}
	m, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, mr := range m.MRs {
		if mr.IID == gone {
			t.Fatalf("!%s merged but is still in the mirror", gone)
		}
	}
	if after.MRsFetched != 0 {
		t.Errorf("a row leaving the manifest is not a reason to fetch anything: %d fetched",
			after.MRsFetched)
	}
}

func iidOf(t *testing.T, node json.RawMessage) string {
	t.Helper()
	var st struct {
		IID string `json:"iid"`
	}
	if err := json.Unmarshal(node, &st); err != nil {
		t.Fatal(err)
	}
	return st.IID
}

// bumpUpdatedAt is a row changing on the forge, which is all the manifest reports.
func bumpUpdatedAt(t *testing.T, node json.RawMessage) struct {
	iid  string
	node json.RawMessage
} {
	t.Helper()
	var row map[string]any
	if err := json.Unmarshal(node, &row); err != nil {
		t.Fatal(err)
	}
	row["updatedAt"] = "2099-01-01T00:00:00Z"
	b, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	return struct {
		iid  string
		node json.RawMessage
	}{iid: iidOf(t, node), node: b}
}
