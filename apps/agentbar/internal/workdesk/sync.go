package workdesk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Fetcher is the part of the GitLab client sync needs. Declared here, as the consumer,
// so the sync can be tested with a fake and so this package does not depend on how the
// forge is reached.
type Fetcher interface {
	Host(ctx context.Context) (string, error)
	User(ctx context.Context) (string, error)
	MergeRequestStamps(ctx context.Context, project, who string) ([]json.RawMessage, error)
	IssueStamps(ctx context.Context, project, who string) ([]json.RawMessage, error)
	MergeRequestsByIID(ctx context.Context, project string, iids []string) ([]json.RawMessage, error)
	IssuesByIID(ctx context.Context, project string, iids []string) ([]json.RawMessage, error)
	Todos(ctx context.Context, project string, actions []string) ([]json.RawMessage, error)
}

// stamp is one row's identity and its change token, which is all a manifest carries.
type stamp struct {
	IID       string `json:"iid"`
	UpdatedAt string `json:"updatedAt"`
}

// SyncResult is what a sync did, for the caller to report. Refreshed counts the rows
// that were actually fetched in full - the rest were already in the mirror and correct.
type SyncResult struct {
	Project       string
	User          string
	MRs           int
	Issues        int
	Todos         int
	MRsFetched    int
	IssuesFetched int
	Took          time.Duration
}

// Sync replaces the mirror with a fresh full snapshot.
//
// The snapshot is still full, and that is what makes a merge request that merged, or an
// issue that closed, disappear for free. What changed is how it is assembled: a manifest
// call names every open row and carries GitLab's updatedAt for each, and only the rows
// whose token moved are fetched in full. The manifest is the authority on what exists,
// so nothing lingers, and there is no cursor state to drift out of step with the forge.
//
// Why it is worth the machinery: a detail node costs about 0.4s of GitLab's time, and a
// manifest of a whole queue costs 0.4s once. A sync with nothing new is a single call.
//
// Written to a staging directory and moved into place at the end, so a sync that fails
// halfway leaves the previous mirror intact rather than a truncated one that still looks
// whole.
func Sync(ctx context.Context, f Fetcher, dir, project string, now time.Time) (*SyncResult, error) {
	return SyncWithProgress(ctx, f, dir, project, now, nil)
}

// Progress is told a leg's name as it starts and again when it lands, with how many rows
// it brought back. A sync is seconds of network with the UI torn down, so the caller has
// something to put on screen; nil reports nothing. It is called from the fetch
// goroutines, so an implementation has to be safe for concurrent use.
type Progress func(leg string, done bool, n int)

func (p Progress) report(leg string, done bool, n int) {
	if p != nil {
		p(leg, done, n)
	}
}

// SyncWithProgress is Sync, reporting each leg as it goes.
func SyncWithProgress(ctx context.Context, f Fetcher, dir, project string, now time.Time,
	p Progress) (*SyncResult, error) {
	started := now
	p.report("identity", false, 0)
	who, err := f.User(ctx)
	if err != nil {
		return nil, fmt.Errorf("identify: %w", err)
	}
	p.report("identity", true, 0)

	// Best effort: no mirror yet, or an unreadable one, simply means every row is new.
	prev, _ := Load(dir)
	if prev == nil {
		prev = &Mirror{}
	}

	// The three legs are independent, so they run together.
	var (
		mrs               []MergeRequest
		issues            []Issue
		todos             []Todo
		mrsGot, issuesGot int
		mrsErr, issuesErr error
		wg                sync.WaitGroup
	)
	p.report("merge requests", false, 0)
	p.report("issues", false, 0)
	p.report("todos", false, 0)
	wg.Go(func() {
		mrs, mrsGot, mrsErr = refresh(ctx, project, who, prev.MRs, mrKey,
			f.MergeRequestStamps, f.MergeRequestsByIID)
		p.report("merge requests", true, len(mrs))
	})
	wg.Go(func() {
		issues, issuesGot, issuesErr = refresh(ctx, project, who, prev.Issues, issueKey,
			f.IssueStamps, f.IssuesByIID)
		p.report("issues", true, len(issues))
	})
	wg.Go(func() {
		// A todo failure is not fatal: a token without the scope simply has none, and
		// the inferred bands still work without it.
		if raw, err := f.Todos(ctx, project, TodoActions()); err == nil {
			_ = decodeInto(raw, &todos)
		}
		p.report("todos", true, len(todos))
	})
	wg.Wait()

	if err := errors.Join(mrsErr, issuesErr); err != nil {
		return nil, err
	}

	m := &Mirror{
		MRs:    mrs,
		Issues: issues,
		Todos:  todos,
		Meta: Meta{
			Project: project,
			User:    who,
			Synced:  started.Format(SyncedLayout),
		},
	}

	p.report("writing", false, 0)
	if err := WriteMirror(dir, m, started); err != nil {
		return nil, err
	}
	p.report("writing", true, 0)
	return &SyncResult{
		Project: project, User: who,
		MRs: len(m.MRs), Issues: len(m.Issues), Todos: len(m.Todos),
		MRsFetched: mrsGot, IssuesFetched: issuesGot,
		Took: time.Since(started),
	}, nil
}

func mrKey(m MergeRequest) (iid, updated string) { return m.IID, m.UpdatedAt }

func issueKey(i Issue) (iid, updated string) { return i.IID, i.UpdatedAt }

// refresh assembles one collection of the mirror.
//
// The manifest is the authority twice over: it says which rows are open - so a merged
// merge request falls out with nothing to clean up - and it carries the token that says
// which of them moved. Only those are fetched in full; the rest are the rows already on
// disk, still correct because GitLab says they have not changed.
//
// A row the manifest names but the detail fetch does not return is dropped rather than
// kept: it merged or closed between the two calls, and the count reported is what was
// actually assembled, so nothing claims to be complete when it is not.
func refresh[T any](ctx context.Context, project, who string, prev []T,
	key func(T) (iid, updated string),
	manifest func(ctx context.Context, project, who string) ([]json.RawMessage, error),
	detail func(ctx context.Context, project string, iids []string) ([]json.RawMessage, error),
) ([]T, int, error) {
	raw, err := manifest(ctx, project, who)
	if err != nil {
		return nil, 0, err
	}
	var stamps []stamp
	if err := decodeInto(raw, &stamps); err != nil {
		return nil, 0, fmt.Errorf("manifest: %w", err)
	}

	have := make(map[string]T, len(prev))
	token := make(map[string]string, len(prev))
	for _, row := range prev {
		iid, updated := key(row)
		have[iid] = row
		token[iid] = updated
	}

	var want []string
	for _, st := range stamps {
		if was, known := token[st.IID]; !known || was != st.UpdatedAt {
			want = append(want, st.IID)
		}
	}

	fetched := 0
	if len(want) > 0 {
		nodes, err := detail(ctx, project, want)
		if err != nil {
			return nil, 0, err
		}
		var fresh []T
		if err := decodeInto(nodes, &fresh); err != nil {
			return nil, 0, err
		}
		for _, row := range fresh {
			iid, _ := key(row)
			have[iid] = row
		}
		fetched = len(fresh)
	}

	rows := make([]T, 0, len(stamps))
	for _, st := range stamps {
		if row, ok := have[st.IID]; ok {
			rows = append(rows, row)
		}
	}
	return rows, fetched, nil
}

func decodeInto[T any](raw []json.RawMessage, out *[]T) error {
	for i, r := range raw {
		var v T
		if err := json.Unmarshal(r, &v); err != nil {
			return fmt.Errorf("node %d: %w", i, err)
		}
		*out = append(*out, v)
	}
	return nil
}

// WriteMirror puts a snapshot and everything derived from it on disk, atomically.
//
// The derived tiers - the index and the pre-rendered documents - are written here rather
// than lazily, so no interactive command ever pays to build them. `Render` rebuilds them
// from a snapshot already on disk when a template changes.
func WriteMirror(dir string, m *Mirror, now time.Time) error {
	stage, err := os.MkdirTemp(filepath.Dir(dir), ".workdesk-stage-*")
	if err != nil {
		return fmt.Errorf("stage dir: %w", err)
	}
	defer os.RemoveAll(stage)

	for _, f := range []struct {
		name string
		data any
	}{
		{"mrs.json", m.MRs},
		{"issues.json", m.Issues},
		{"todos.json", m.Todos},
		{"meta.json", m.Meta},
		{"index.json", BuildIndex(m)},
	} {
		if err := writeJSON(filepath.Join(stage, f.name), f.data); err != nil {
			return err
		}
	}
	if err := renderInto(stage, m, now); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mirror dir: %w", err)
	}
	// Document directories are replaced wholesale, never merged into: a merge request
	// that merged since the last sync has to leave, and that only happens if the old
	// directory goes.
	for _, sub := range []string{"mr", "issue"} {
		if err := os.RemoveAll(filepath.Join(dir, sub)); err != nil {
			return fmt.Errorf("clear %s: %w", sub, err)
		}
	}
	entries, err := os.ReadDir(stage)
	if err != nil {
		return fmt.Errorf("read stage: %w", err)
	}
	for _, e := range entries {
		from := filepath.Join(stage, e.Name())
		to := filepath.Join(dir, e.Name())
		if err := os.RemoveAll(to); err != nil {
			return fmt.Errorf("replace %s: %w", e.Name(), err)
		}
		if err := os.Rename(from, to); err != nil {
			return fmt.Errorf("move %s into place: %w", e.Name(), err)
		}
	}
	return nil
}

// Render rebuilds the derived tiers from the snapshot already on disk, with no network.
// Split out of Sync so a fixture mirror can exist without a forge behind it, and so a
// change to a template does not mean re-fetching the whole queue.
func Render(dir string, now time.Time) (*Mirror, error) {
	m, err := Load(dir)
	if err != nil {
		return nil, err
	}
	return m, WriteMirror(dir, m, now)
}

func renderInto(dir string, m *Mirror, now time.Time) error {
	if err := os.MkdirAll(filepath.Join(dir, "mr"), 0o755); err != nil {
		return fmt.Errorf("mr dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "issue"), 0o755); err != nil {
		return fmt.Errorf("issue dir: %w", err)
	}
	board, err := os.Create(filepath.Join(dir, "board.md"))
	if err != nil {
		return fmt.Errorf("board.md: %w", err)
	}
	if err := Board(board, m, now); err != nil {
		board.Close()
		return fmt.Errorf("render board: %w", err)
	}
	if err := board.Close(); err != nil {
		return fmt.Errorf("board.md: %w", err)
	}
	for i := range m.MRs {
		mr := &m.MRs[i]
		if err := writeDoc(filepath.Join(dir, "mr", mr.IID+".md"), func(f *os.File) error {
			return Sheet(f, mr, m.Meta, now)
		}); err != nil {
			return fmt.Errorf("render !%s: %w", mr.IID, err)
		}
	}
	for i := range m.Issues {
		is := &m.Issues[i]
		if err := writeDoc(filepath.Join(dir, "issue", is.IID+".md"), func(f *os.File) error {
			return IssueSheet(f, is)
		}); err != nil {
			return fmt.Errorf("render #%s: %w", is.IID, err)
		}
	}
	return nil
}

func writeDoc(path string, render func(*os.File) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := render(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func writeJSON(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}

// LoadIndex reads the small tier. Every interactive command uses this rather than Load:
// it is the difference between a popup that opens instantly and one that decodes
// megabytes on every cursor movement.
func LoadIndex(dir string) (*Index, error) {
	b, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("index.json: %w", ErrNoMirror)
		}
		return nil, fmt.Errorf("read index.json: %w", err)
	}
	var idx Index
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, fmt.Errorf("decode index.json: %w", err)
	}
	return &idx, nil
}
