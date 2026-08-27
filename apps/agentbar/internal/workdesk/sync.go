package workdesk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Fetcher is the part of the GitLab client sync needs. Declared here, as the consumer,
// so the sync can be tested with a fake and so this package does not depend on how the
// forge is reached.
type Fetcher interface {
	Host(ctx context.Context) (string, error)
	User(ctx context.Context) (string, error)
	MergeRequests(ctx context.Context, project, who string) ([]json.RawMessage, error)
	Issues(ctx context.Context, project, who string) ([]json.RawMessage, error)
	Todos(ctx context.Context, project string) ([]json.RawMessage, error)
}

// SyncResult is what a sync did, for the caller to report.
type SyncResult struct {
	Project string
	User    string
	MRs     int
	Issues  int
	Todos   int
	Took    time.Duration
}

// Sync replaces the mirror with a fresh full snapshot.
//
// A full snapshot every time, never an incremental update: that is what makes a merge
// request that merged, or an issue that closed, disappear for free, with no cursor state
// to drift out of step with the forge.
//
// Written to a staging directory and moved into place at the end, so a sync that fails
// halfway leaves the previous mirror intact rather than a truncated one that still looks
// whole.
func Sync(ctx context.Context, f Fetcher, dir, project string, now time.Time) (*SyncResult, error) {
	started := now
	who, err := f.User(ctx)
	if err != nil {
		return nil, fmt.Errorf("identify: %w", err)
	}

	// The three fetches are independent, so they run together. Merge request pages
	// stay sequential inside their own fetch - they are cursor-chained.
	type result struct {
		mrs, issues, todos []json.RawMessage
		err                error
	}
	mrsCh := make(chan result, 1)
	issuesCh := make(chan result, 1)
	todosCh := make(chan result, 1)

	go func() {
		n, err := f.MergeRequests(ctx, project, who)
		mrsCh <- result{mrs: n, err: err}
	}()
	go func() {
		n, err := f.Issues(ctx, project, who)
		issuesCh <- result{issues: n, err: err}
	}()
	go func() {
		// A todo failure is not fatal: a token without the scope simply has none, and
		// the inferred bands still work without it.
		n, err := f.Todos(ctx, project)
		if err != nil {
			n = nil
		}
		todosCh <- result{todos: n}
	}()

	mrsRes, issuesRes, todosRes := <-mrsCh, <-issuesCh, <-todosCh
	if err := errors.Join(mrsRes.err, issuesRes.err); err != nil {
		return nil, err
	}

	m := &Mirror{Meta: Meta{
		Project: project,
		User:    who,
		Synced:  started.Format("2006-01-02 15:04:05"),
	}}
	if err := decodeInto(mrsRes.mrs, &m.MRs); err != nil {
		return nil, fmt.Errorf("merge requests: %w", err)
	}
	if err := decodeInto(issuesRes.issues, &m.Issues); err != nil {
		return nil, fmt.Errorf("issues: %w", err)
	}
	if err := decodeInto(todosRes.todos, &m.Todos); err != nil {
		return nil, fmt.Errorf("todos: %w", err)
	}

	if err := WriteMirror(dir, m, started); err != nil {
		return nil, err
	}
	return &SyncResult{
		Project: project, User: who,
		MRs: len(m.MRs), Issues: len(m.Issues), Todos: len(m.Todos),
		Took: time.Since(started),
	}, nil
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
