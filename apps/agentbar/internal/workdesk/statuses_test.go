package workdesk

import (
	"strings"
	"testing"
	"time"
)

// lifecycle is a project's statuses as GitLab declares them: the work in order, the
// finished categories last.
func lifecycle() []Status {
	return []Status{
		{ID: "s1", Name: "Backlog", Category: "triage"},
		{ID: "s2", Name: "To do", Category: "to_do"},
		{ID: "s3", Name: "In progress", Category: "in_progress"},
		{ID: "s4", Name: "Done", Category: "done"},
		{ID: "s5", Name: "Cancelled", Category: "canceled"},
	}
}

func TestLifecycleOrdersAndFlagsStatuses(t *testing.T) {
	t.Parallel()
	l := NewLifecycle(lifecycle())
	cases := []struct {
		status string
		rank   int
		active bool
	}{
		{"Backlog", 0, true},
		{"To do", 1, true},
		{"In progress", 2, true},
		{"Done", 3, false},
		{"Cancelled", 4, false},
		// A status the lifecycle no longer carries, and an issue GitLab has none for:
		// both sort after everything known rather than joining the first band.
		{"Retired", 5, false},
		{NoStatus, 5, false},
	}
	for _, c := range cases {
		t.Run(c.status, func(t *testing.T) {
			t.Parallel()
			if got := l.Rank(c.status); got != c.rank {
				t.Errorf("Rank(%q) = %d, want %d", c.status, got, c.rank)
			}
			if got := l.Active(c.status); got != c.active {
				t.Errorf("Active(%q) = %v, want %v", c.status, got, c.active)
			}
		})
	}
}

// The UI draws one line where the flag flips, so the flag has to flip exactly once
// however the lifecycle is arranged.
func TestLifecycleFlagsFlipOnce(t *testing.T) {
	t.Parallel()
	idx := BuildIndex(statusMirror())
	flips := 0
	prev := "a"
	for _, it := range idx.Issues {
		if it.Flag != prev {
			flips++
			prev = it.Flag
		}
	}
	if flips != 1 {
		t.Errorf("the active/inactive flag flipped %d times, want exactly 1", flips)
	}
}

// statusMirror is one issue per status plus one with none, deliberately out of order, so
// the banding is what puts them in it.
func statusMirror() *Mirror {
	sprint := &Iteration{ID: "it1", StartDate: "2026-08-24", DueDate: "2026-09-06"}
	issue := func(iid, status string, in bool, labels ...string) Issue {
		is := Issue{IID: iid, Title: "issue " + iid, UpdatedAt: "2026-08-27T09:00:00Z"}
		for _, st := range lifecycle() {
			if st.Name == status {
				is.Status = &Status{ID: st.ID, Name: st.Name, Category: st.Category}
			}
		}
		if in {
			is.Iteration = sprint
		}
		for _, l := range labels {
			is.Labels.Nodes = append(is.Labels.Nodes, Label{Title: l})
		}
		return is
	}
	return &Mirror{
		Issues: []Issue{
			issue("5", "Cancelled", false),
			issue("4", "Done", false),
			issue("6", "", false),
			issue("2", "To do", true, "prio::high", "flaky"),
			issue("1", "Backlog", false),
			issue("3", "In progress", true),
		},
		Meta: Meta{Statuses: lifecycle(), Iteration: sprint},
	}
}

// The bands are GitLab's columns in GitLab's order, and an issue it has no status for
// still gets a row.
func TestIssuesBandByStatusInLifecycleOrder(t *testing.T) {
	t.Parallel()
	idx := BuildIndex(statusMirror())
	var got []string
	for _, it := range idx.Issues {
		got = append(got, it.Label+" #"+strings.TrimPrefix(it.Ref, "#"))
	}
	want := "Backlog #1, To do #2, In progress #3, Done #4, Cancelled #5, no status #6"
	if strings.Join(got, ", ") != want {
		t.Errorf("issue bands\n got %s\nwant %s", strings.Join(got, ", "), want)
	}
}

func TestIssueRowsCarryTagsAndSprint(t *testing.T) {
	t.Parallel()
	rows := BuildIndex(statusMirror()).Rows(ViewIssues, time.Now())
	for _, r := range rows {
		switch r.Ref {
		case "#2":
			// Scoped labels show their value alone; a plain one shows whole.
			if got := strings.Join(r.Tags, ","); got != "high,flaky" {
				t.Errorf("#2 tags = %q, want \"high,flaky\"", got)
			}
			if !r.Sprint {
				t.Error("#2 is in the sprint but the row does not say so")
			}
		case "#1":
			if r.Sprint {
				t.Error("#1 is not in the sprint but the row says it is")
			}
		}
	}
}

// An issue in some other iteration is not in this one, and a project with no sprint
// marks nothing.
func TestInSprintComparesTheCurrentIteration(t *testing.T) {
	t.Parallel()
	current := &Iteration{ID: "it1"}
	cases := []struct {
		name string
		is   Issue
		cur  *Iteration
		want bool
	}{
		{"in it", Issue{Iteration: &Iteration{ID: "it1"}}, current, true},
		{"in another one", Issue{Iteration: &Iteration{ID: "it0"}}, current, false},
		{"in none", Issue{}, current, false},
		{"no current sprint", Issue{Iteration: &Iteration{ID: "it1"}}, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.is.InSprint(c.cur); got != c.want {
				t.Errorf("InSprint = %v, want %v", got, c.want)
			}
		})
	}
}

// A project without the licensed features has no lifecycle, and every issue still gets a
// row rather than the view going blank.
func TestNoLifecycleStillRendersEveryIssue(t *testing.T) {
	t.Parallel()
	m := statusMirror()
	m.Meta.Statuses = nil
	m.Meta.Iteration = nil
	idx := BuildIndex(m)
	if len(idx.Issues) != 6 {
		t.Fatalf("%d issue rows, want 6", len(idx.Issues))
	}
	for _, it := range idx.Issues {
		if it.Flag != "i" {
			t.Errorf("%s is flagged %q with no lifecycle to say it is asking anything", it.Ref, it.Flag)
		}
	}
}
