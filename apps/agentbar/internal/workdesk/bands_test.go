package workdesk

import (
	"strings"
	"testing"
	"time"
)

// mr builds a merge request that is green, reviewed and mergeable, so each case below
// states only what it is testing. Every band therefore reads as one deviation from a
// merge request that would otherwise land itself.
func mr(edit func(*MergeRequest)) *MergeRequest {
	m := &MergeRequest{
		IID:          "1",
		Title:        "a change",
		SourceBranch: "feat/a",
		TargetBranch: "main",
		UpdatedAt:    "2026-08-27T09:00:00Z",
		HeadPipeline: &Pipeline{Status: "SUCCESS"},
	}
	m.HeadPipeline.DetailedStatus.Label = "passed"
	m.Reviewers.Nodes = []Reviewer{{Username: "dana"}}
	m.Reviewers.Nodes[0].Interaction.ReviewState = "APPROVED"
	if edit != nil {
		edit(m)
	}
	return m
}

func failing(ids ...string) []Check {
	out := make([]Check, 0, len(ids))
	for _, id := range ids {
		out = append(out, Check{Identifier: id, Status: "FAILED"})
	}
	return out
}

func TestBand(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		mr   *MergeRequest
		want Band
	}{{
		name: "green and reviewed, auto-merge never set",
		mr:   mr(nil),
		want: BandAutoMerge,
	}, {
		name: "green and reviewed, auto-merge set: it lands itself",
		mr:   mr(func(m *MergeRequest) { m.AutoMergeEnabled = true }),
		want: BandAutoMerging,
	}, {
		name: "a reviewer asked for changes",
		mr: mr(func(m *MergeRequest) {
			m.Reviewers.Nodes[0].Interaction.ReviewState = "REQUESTED_CHANGES"
		}),
		want: BandReturned,
	}, {
		name: "pipeline is red",
		mr:   mr(func(m *MergeRequest) { m.HeadPipeline.Status = "FAILED" }),
		want: BandCI,
	}, {
		name: "conflicts with the target branch",
		mr: mr(func(m *MergeRequest) {
			m.Conflicts = true
			m.MergeabilityChecks = failing("CONFLICT")
		}),
		want: BandStuck,
	}, {
		name: "review threads left open",
		mr:   mr(func(m *MergeRequest) { m.MergeabilityChecks = failing("DISCUSSIONS_NOT_RESOLVED") }),
		want: BandStuck,
	}, {
		name: "finished, nobody was ever asked to look",
		mr:   mr(func(m *MergeRequest) { m.Reviewers.Nodes = nil }),
		want: BandUnasked,
	}, {
		name: "assigned and genuinely waiting on someone else",
		mr:   mr(func(m *MergeRequest) { m.MergeabilityChecks = failing("NOT_APPROVED") }),
		want: BandApprovals,
	}, {
		name: "draft",
		mr:   mr(func(m *MergeRequest) { m.Draft = true }),
		want: BandDraft,
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.mr.Band(); got != c.want {
				t.Errorf("Band() = %v (%q), want %v (%q)", got, got, c.want, c.want)
			}
		})
	}
}

// The order of the arms is the design, so it gets its own cases: a merge request can
// satisfy several at once and only one band is right.
func TestBandPrecedence(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		mr   *MergeRequest
		want Band
		why  string
	}{{
		name: "draft beats everything",
		mr: mr(func(m *MergeRequest) {
			m.Draft = true
			m.HeadPipeline.Status = "FAILED"
			m.Reviewers.Nodes = nil
		}),
		want: BandDraft,
		why:  "a draft is deliberate; nothing about it is asking for attention yet",
	}, {
		name: "changes requested beats a red pipeline",
		mr: mr(func(m *MergeRequest) {
			m.Reviewers.Nodes[0].Interaction.ReviewState = "REQUESTED_CHANGES"
			m.HeadPipeline.Status = "FAILED"
		}),
		want: BandReturned,
		why:  "a human is waiting on a reply, which outranks a machine",
	}, {
		name: "red pipeline beats having no reviewer",
		mr: mr(func(m *MergeRequest) {
			m.HeadPipeline.Status = "FAILED"
			m.Reviewers.Nodes = nil
		}),
		want: BandCI,
		why:  "there is no point assigning a reviewer to a red branch",
	}, {
		name: "no reviewer beats waiting for approvals",
		mr: mr(func(m *MergeRequest) {
			m.Reviewers.Nodes = nil
			m.MergeabilityChecks = failing("NOT_APPROVED")
		}),
		want: BandUnasked,
		why:  "not approved because nobody was asked is a different problem from not approved yet",
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.mr.Band(); got != c.want {
				t.Errorf("Band() = %q, want %q\n  %s", got, c.want, c.why)
			}
		})
	}
}

func TestBandActiveSplit(t *testing.T) {
	t.Parallel()
	active := []Band{BandReturned, BandCI, BandStuck, BandUnasked, BandAutoMerge}
	inactive := []Band{BandApprovals, BandAutoMerging, BandDraft}

	for _, b := range active {
		if !b.Active() || b.Flag() != "a" {
			t.Errorf("%q should be active, got Active()=%v Flag()=%q", b, b.Active(), b.Flag())
		}
	}
	for _, b := range inactive {
		if b.Active() || b.Flag() != "i" {
			t.Errorf("%q should be inactive, got Active()=%v Flag()=%q", b, b.Active(), b.Flag())
		}
	}
	// Every band needs a label; an unnamed band would render as an empty header.
	for b := BandReturned; b <= BandDraft; b++ {
		if b.String() == "" {
			t.Errorf("band %d has no name", int(b))
		}
	}
}

// GitLab adds mergeability checks over time and does not document the full set, so an
// identifier we have never seen must still reach the reader.
func TestGateMessageUnknownIdentifierDegrades(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"NOT_APPROVED":             "not enough approvals",
		"CI_MUST_PASS":             "pipeline must pass",
		"SOME_FUTURE_GITLAB_CHECK": "some_future_gitlab_check",
	}
	for id, want := range cases {
		if got := GateMessage(id); got != want {
			t.Errorf("GateMessage(%q) = %q, want %q", id, got, want)
		}
	}
}

func TestBlockersReportOnlyFailures(t *testing.T) {
	t.Parallel()
	m := mr(func(m *MergeRequest) {
		m.MergeabilityChecks = []Check{
			{Identifier: "NOT_APPROVED", Status: "FAILED"},
			{Identifier: "CI_MUST_PASS", Status: "SUCCESS"},
			// INACTIVE means the gate is not configured on this project: silence,
			// not a pass, and never a blocker.
			{Identifier: "JIRA_ASSOCIATION_MISSING", Status: "INACTIVE"},
			{Identifier: "CONFLICT", Status: "CHECKING"},
		}
	})
	if got := strings.Join(m.Blockers(), ", "); got != "not enough approvals" {
		t.Errorf("Blockers() = %q, want only the FAILED gate", got)
	}
	if got := strings.Join(m.Pending(), ", "); got != "conflicts with the target branch" {
		t.Errorf("Pending() = %q, want only the CHECKING gate", got)
	}
}

// A merge request with no pipeline decodes to a nil pointer. Reading through it must
// not panic, and must not read as a pipeline that reported nothing.
func TestNoPipelineIsNotAFailedPipeline(t *testing.T) {
	t.Parallel()
	m := mr(func(m *MergeRequest) { m.HeadPipeline = nil })
	if got := m.CIStatus(); got != "NONE" {
		t.Errorf("CIStatus() = %q, want NONE", got)
	}
	if got := m.PipelineLabel(); got != "no pipeline" {
		t.Errorf("PipelineLabel() = %q, want %q", got, "no pipeline")
	}
	if got := m.Band(); got != BandAutoMerge {
		t.Errorf("Band() = %q; a missing pipeline must not read as a failed one", got)
	}
}

func TestIssuePriorityHighestLabelWins(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		labels []string
		want   Priority
	}{
		{"no labels at all", nil, PrioNone},
		{"only unrelated labels", []string{"type::bug"}, PrioNone},
		{"high", []string{"prio::high"}, PrioHigh},
		{"mid among others", []string{"type::bug", "prio::mid"}, PrioMid},
		{"low", []string{"prio::low"}, PrioLow},
		{"several: the highest wins, not the first returned", []string{"prio::low", "prio::high"}, PrioHigh},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var i Issue
			for _, l := range c.labels {
				i.Labels.Nodes = append(i.Labels.Nodes, Label{Title: l})
			}
			if got := i.Priority(); got != c.want {
				t.Errorf("Priority() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestPadAndShortCountRunesNotBytes(t *testing.T) {
	t.Parallel()
	// The bug this closes: awk's %-Ns pads by bytes, and the ellipsis is three of
	// them, so every truncated row came out three columns short.
	long := "Registry client should honour Retry-After headers"
	got := Pad(long, 40)
	if n := len([]rune(got)); n != 40 {
		t.Errorf("Pad(%q, 40) is %d runes, want 40", got, n)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("Pad(%q, 40) should mark the cut with an ellipsis", got)
	}
	if got := Pad("short", 10); got != "short     " {
		t.Errorf("Pad(%q) = %q, want right-padding to 10", "short", got)
	}
	if got := Short("exact", 5); got != "exact" {
		t.Errorf("Short at exactly n should not truncate, got %q", got)
	}
}

func TestAge(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name, stamp, want, wantAgo string
	}{
		{"this morning", "2026-08-27T09:00:00Z", "today", "today"},
		{"yesterday", "2026-08-26T09:00:00Z", "1d", "1d ago"},
		{"nine days", "2026-08-18T11:00:00Z", "9d", "9d ago"},
		{"unparseable", "not a date", "", ""},
		// Clock skew against the forge is not information: it reads as today
		// rather than as a negative day count.
		{"in the future", "2026-08-29T09:00:00Z", "today", "today"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ts := ParseTime(c.stamp)
			if got := Age(ts, now); got != c.want {
				t.Errorf("Age(%q) = %q, want %q", c.stamp, got, c.want)
			}
			if got := AgeAgo(ts, now); got != c.wantAgo {
				t.Errorf("AgeAgo(%q) = %q, want %q", c.stamp, got, c.wantAgo)
			}
		})
	}
}

func TestTSVEscapesSeparators(t *testing.T) {
	t.Parallel()
	got := TSV("band", "a title\twith a tab", "and\na newline")
	if strings.Count(got, "\t") != 2 {
		t.Errorf("TSV(%q) has %d real tabs, want 2 - a title must not invent a column",
			got, strings.Count(got, "\t"))
	}
	if strings.Contains(got, "\n") {
		t.Errorf("TSV(%q) contains a real newline - a title must not invent a row", got)
	}
}
