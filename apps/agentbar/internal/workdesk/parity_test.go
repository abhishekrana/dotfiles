package workdesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// frozen is the instant the golden files were captured from the shell implementation
// this package replaces. Ages are relative to now, so the goldens are only meaningful
// against the clock that produced them - without this they would drift a day at every
// midnight and the parity tests would rot.
func frozen(t *testing.T) time.Time {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "captured-at"))
	if err != nil {
		t.Fatalf("read captured-at: %v", err)
	}
	ts, err := time.Parse(time.RFC3339, strings.TrimSpace(string(b)))
	if err != nil {
		t.Fatalf("parse captured-at: %v", err)
	}
	return ts
}

func loadFixture(t *testing.T) *Mirror {
	t.Helper()
	m, err := FixtureMirror()
	if err != nil {
		t.Fatalf("load fixture mirror: %v", err)
	}
	return m
}

func golden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return string(b)
}

// The fixture covers one merge request per band, so decoding it and classifying every
// row proves Band() against real API shapes rather than hand-built structs.
func TestBandsMatchGolden(t *testing.T) {
	t.Parallel()
	m := loadFixture(t)

	// band name -> the iids the shell put in it, read out of its own output so the
	// expectation cannot drift from the file it is checked against.
	want := map[string][]string{}
	for _, line := range strings.Split(strings.TrimRight(golden(t, "list-mrs.golden"), "\n"), "\n") {
		f := strings.Split(line, "\t")
		if len(f) < 3 {
			t.Fatalf("golden row has %d fields: %q", len(f), line)
		}
		want[f[0]] = append(want[f[0]], strings.TrimPrefix(f[2], "!"))
	}

	got := map[string][]string{}
	byIID := map[string]*MergeRequest{}
	for i := range m.MRs {
		mr := &m.MRs[i]
		byIID[mr.IID] = mr
		got[mr.Band().String()] = append(got[mr.Band().String()], mr.IID)
	}

	if len(m.MRs) == 0 {
		t.Fatal("fixture decoded zero merge requests")
	}
	for band, iids := range want {
		for _, iid := range iids {
			mr, ok := byIID[iid]
			if !ok {
				t.Errorf("!%s is in the golden but not in the decoded fixture", iid)
				continue
			}
			if b := mr.Band().String(); b != band {
				t.Errorf("!%s: Band() = %q, shell said %q (blockers: %v, ci: %s, reviewers: %d, auto: %v)",
					iid, b, band, mr.Blockers(), mr.CIStatus(), len(mr.Reviewers.Nodes), mr.AutoMergeEnabled)
			}
		}
	}
	// All eight bands should be exercised; a fixture that stopped covering one would
	// let a regression through unnoticed.
	if len(want) != 8 {
		t.Errorf("fixture covers %d bands, want all 8: %v", len(want), keysOf(want))
	}
}

func keysOf(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// The decode has to survive the real shapes: string iids from GraphQL, numeric iids
// from the todos REST endpoint, and a nil pipeline.
func TestFixtureDecodes(t *testing.T) {
	t.Parallel()
	m := loadFixture(t)
	if m.Meta.Project == "" {
		t.Error("meta.project did not decode")
	}
	if len(m.Issues) == 0 {
		t.Error("no issues decoded")
	}
	if len(m.Todos) == 0 {
		t.Error("no todos decoded")
	}
	for _, td := range m.Todos {
		if td.Target.IID.String() == "" {
			t.Errorf("todo %d: numeric target.iid did not decode through flexID", td.ID)
		}
	}
	for _, mr := range m.MRs {
		if mr.IID == "" || mr.Title == "" {
			t.Errorf("merge request decoded with empty iid or title: %+v", mr.IID)
		}
	}
}

// The shell implementation this package replaces is the specification, and its output
// is frozen in testdata. Byte-identical is the bar: a difference of one space is a
// column that no longer lines up.
func TestListRowsMatchGoldenExactly(t *testing.T) {
	t.Parallel()
	now := frozen(t)
	idx := BuildIndex(loadFixture(t))

	for _, c := range []struct {
		view   View
		golden string
	}{
		{ViewMRs, "list-mrs.golden"},
		{ViewIssues, "list-issues.golden"},
		{ViewInbox, "list-inbox.golden"},
	} {
		t.Run(c.view.String(), func(t *testing.T) {
			t.Parallel()
			var got strings.Builder
			if err := WriteRows(&got, idx.Rows(c.view, now)); err != nil {
				t.Fatalf("WriteRows: %v", err)
			}
			diffLines(t, golden(t, c.golden), got.String())
		})
	}
}

// diffLines reports the first differing line and its neighbourhood. A whole-file dump
// on failure is unreadable when the files are dozens of tab-separated rows.
func diffLines(t *testing.T, want, got string) {
	t.Helper()
	if want == got {
		return
	}
	wl := strings.Split(strings.TrimRight(want, "\n"), "\n")
	gl := strings.Split(strings.TrimRight(got, "\n"), "\n")
	for i := 0; i < len(wl) || i < len(gl); i++ {
		w, g := "<missing>", "<missing>"
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w == g {
			continue
		}
		t.Errorf("line %d differs\n  want %s\n  got  %s", i+1,
			strings.ReplaceAll(w, "\t", "→"), strings.ReplaceAll(g, "\t", "→"))
		if len(wl) != len(gl) {
			t.Errorf("(%d lines want, %d got)", len(wl), len(gl))
		}
		return
	}
}

// The picker's own row format - band headers, the active/inactive divider, and the
// visible column - checked against the shell byte for byte. A column that shifts by one
// is a list that no longer lines up.
func TestPickerRowsMatchGoldenExactly(t *testing.T) {
	t.Parallel()
	now := frozen(t)
	m := loadFixture(t)
	idx := BuildIndex(m)

	agents, err := fixtureAgents(t)
	if err != nil {
		t.Fatalf("fixture agents: %v", err)
	}

	for _, c := range []struct {
		view   View
		rows   []Row
		golden string
	}{
		{ViewMRs, idx.Rows(ViewMRs, now), "rows-mrs.golden"},
		{ViewIssues, idx.Rows(ViewIssues, now), "rows-issues.golden"},
		{ViewInbox, idx.Rows(ViewInbox, now), "rows-inbox.golden"},
		{ViewAgents, AgentRows(agents, idx), "rows-agents.golden"},
	} {
		t.Run(c.view.String(), func(t *testing.T) {
			t.Parallel()
			got := strings.Join(PickerRows(c.rows, c.view, "__band__"), "\n") + "\n"
			diffLines(t, golden(t, c.golden), got)
		})
	}
}

// fixtureAgents writes the embedded agent fixture out and reads it back through the same
// parser the mockup uses, so the file format is covered rather than bypassed.
func fixtureAgents(t *testing.T) ([]Agent, error) {
	t.Helper()
	b, err := FixtureAgents()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(t.TempDir(), "agents.tsv")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return nil, err
	}
	return LoadAgents(path)
}
