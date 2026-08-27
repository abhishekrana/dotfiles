package workdesk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The board's prose names the command that renders it, and the command was renamed
// from `work` to `workdesk`. That is the one intended difference from the frozen shell
// output, so it is substituted rather than silently tolerated - anything else that
// moved is a bug.
func boardGolden(t *testing.T) string {
	t.Helper()
	return strings.Replace(golden(t, "board.golden"), "`work mr <iid>`", "`workdesk mr <iid>`", 1)
}

func TestBoardMatchesGolden(t *testing.T) {
	t.Parallel()
	var got strings.Builder
	if err := Board(&got, loadFixture(t), frozen(t)); err != nil {
		t.Fatalf("Board: %v", err)
	}
	diffLines(t, boardGolden(t), got.String())
}

func TestMatrixMatchesGolden(t *testing.T) {
	t.Parallel()
	var got strings.Builder
	if err := Matrix(&got, loadFixture(t)); err != nil {
		t.Fatalf("Matrix: %v", err)
	}
	diffLines(t, golden(t, "matrix.golden"), got.String())
}

// Every merge request in the fixture has a sheet, and each covers something different:
// no pipeline, no reviewers, conflicts, a CODEOWNERS rule nobody met, resolved threads.
// Checking all nine is what makes the sheet render trustworthy rather than checking one.
func TestSheetsMatchGolden(t *testing.T) {
	t.Parallel()
	m := loadFixture(t)
	now := frozen(t)

	entries, err := os.ReadDir(filepath.Join("testdata", "mr"))
	if err != nil {
		t.Fatalf("read testdata/mr: %v", err)
	}
	if len(entries) != len(m.MRs) {
		t.Fatalf("%d golden sheets for %d merge requests", len(entries), len(m.MRs))
	}

	for i := range m.MRs {
		mr := &m.MRs[i]
		t.Run("!"+mr.IID, func(t *testing.T) {
			t.Parallel()
			var got strings.Builder
			if err := Sheet(&got, mr, m.Meta, now); err != nil {
				t.Fatalf("Sheet: %v", err)
			}
			diffLines(t, golden(t, filepath.Join("mr", mr.IID+".md")), got.String())
		})
	}
}

func TestIssueSheetMatchesGolden(t *testing.T) {
	t.Parallel()
	m := loadFixture(t)
	var target *Issue
	for i := range m.Issues {
		if m.Issues[i].IID == "128" {
			target = &m.Issues[i]
		}
	}
	if target == nil {
		t.Fatal("fixture no longer contains issue #128")
	}
	var got strings.Builder
	if err := IssueSheet(&got, target); err != nil {
		t.Fatalf("IssueSheet: %v", err)
	}
	diffLines(t, golden(t, "preview-issue-128.golden"), got.String())
}

// The sheet's job is to answer one question, so the two answers get their own check
// independent of any golden file.
func TestSheetAnswersCanIMergeIt(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		mr    *MergeRequest
		want  string
		avoid string
	}{{
		name:  "nothing refused",
		mr:    mr(nil),
		want:  "## Can I merge it?  **Yes**",
		avoid: "blocker(s)",
	}, {
		name:  "two gates refused",
		mr:    mr(func(m *MergeRequest) { m.MergeabilityChecks = failing("NOT_APPROVED", "CONFLICT") }),
		want:  "## Can I merge it?  **No - 2 blocker(s)**",
		avoid: "**Yes**",
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var got strings.Builder
			if err := Sheet(&got, c.mr, Meta{}, frozen(t)); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(got.String(), c.want) {
				t.Errorf("sheet does not contain %q", c.want)
			}
			if strings.Contains(got.String(), c.avoid) {
				t.Errorf("sheet should not contain %q", c.avoid)
			}
		})
	}
}

// A merge request nobody was asked to review must say so in words, because that is the
// single most common reason work in this queue stops moving.
func TestSheetNamesTheUnassignedCase(t *testing.T) {
	t.Parallel()
	m := mr(func(m *MergeRequest) { m.Reviewers.Nodes = nil })
	var got strings.Builder
	if err := Sheet(&got, m, Meta{}, frozen(t)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.String(), "**Nobody is assigned.**") {
		t.Error("sheet does not call out that nobody is assigned")
	}
}

// The agent preview is the fourth view's whole reason for existing, so both of its shapes
// are checked: work that reached GitLab, and work that did not.
func TestAgentSheetMatchesGolden(t *testing.T) {
	t.Parallel()
	agents, err := fixtureAgents(t)
	if err != nil {
		t.Fatalf("fixture agents: %v", err)
	}
	idx := BuildIndex(loadFixture(t))
	byBranch := map[string]string{}
	for _, it := range idx.MRs {
		byBranch[it.Branch] = it.Ref
	}

	for _, c := range []struct {
		pane, golden string
	}{
		{"%7", "preview-agent-7.golden"},
		{"%2", "preview-agent-2.golden"},
	} {
		t.Run(c.pane, func(t *testing.T) {
			t.Parallel()
			var a Agent
			for _, cand := range agents {
				if cand.Pane == c.pane {
					a = cand
				}
			}
			if a.Pane == "" {
				t.Fatalf("fixture has no pane %s", c.pane)
			}
			var got strings.Builder
			if err := AgentSheet(&got, a, byBranch[a.Branch]); err != nil {
				t.Fatalf("AgentSheet: %v", err)
			}
			// The shell appended the merge request's own sheet after a rule; the
			// preview command still does, so only the agent block is compared here.
			want, _, _ := strings.Cut(golden(t, c.golden), "\n---\n")
			diffLines(t, strings.TrimRight(want, "\n")+"\n", got.String())
		})
	}
}

// An agent that finished with nothing to show for it is the row no forge view can
// produce, so it must say so in words rather than leaving a blank field.
func TestAgentSheetNamesTheMissingMergeRequest(t *testing.T) {
	t.Parallel()
	var got strings.Builder
	a := Agent{State: "done", Title: "a finished agent", Branch: "feat/nothing", Pane: "%1"}
	if err := AgentSheet(&got, a, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.String(), "Nothing in GitLab knows this work") {
		t.Error("the sheet does not say the work never reached GitLab")
	}
}
