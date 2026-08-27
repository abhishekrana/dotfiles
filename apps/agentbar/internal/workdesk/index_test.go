package workdesk

import (
	"encoding/json"
	"strings"
	"testing"
)

// The index is written by sync and read by every interactive command, so a field that
// does not survive the round trip is a row that renders wrong.
func TestIndexRoundTrip(t *testing.T) {
	t.Parallel()
	now := frozen(t)
	built := BuildIndex(loadFixture(t))

	b, err := json.Marshal(built)
	if err != nil {
		t.Fatalf("marshal index: %v", err)
	}
	var back Index
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal index: %v", err)
	}

	for _, v := range []View{ViewInbox, ViewMRs, ViewIssues} {
		t.Run(v.String(), func(t *testing.T) {
			var wantB, gotB strings.Builder
			if err := WriteRows(&wantB, built.Rows(v, now)); err != nil {
				t.Fatal(err)
			}
			if err := WriteRows(&gotB, back.Rows(v, now)); err != nil {
				t.Fatal(err)
			}
			diffLines(t, wantB.String(), gotB.String())
		})
	}
	if back.Generated != built.Generated || back.Project != built.Project {
		t.Errorf("index metadata lost in the round trip: %+v", back.Generated)
	}
}

func TestViewRingWrapsAndParses(t *testing.T) {
	t.Parallel()
	if got := len(Views()); got != 4 {
		t.Fatalf("Views() has %d entries, want 4", got)
	}
	// The ring must close: walking it four times returns where it started, which is
	// what makes tab and 1-4 agree.
	v := ViewInbox
	for range Views() {
		v = v.Next()
	}
	if v != ViewInbox {
		t.Errorf("walking the ring landed on %q, want inbox", v)
	}
	for _, c := range []struct{ in, want string }{
		{"1", "inbox"}, {"2", "mrs"}, {"3", "issues"}, {"4", "agents"},
		{"mrs", "mrs"}, {"agent", "agents"}, {"nonsense", "inbox"}, {"", "inbox"},
	} {
		if got := ParseView(c.in).String(); got != c.want {
			t.Errorf("ParseView(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A todo whose target is neither a merge request nor an issue has no row to sit in.
// The shell assumed anything not an Issue was a merge request and rendered "!0".
func TestTodosOutsideOurViewsAreDropped(t *testing.T) {
	t.Parallel()
	m := &Mirror{Todos: []Todo{
		{State: "pending", TargetType: "MergeRequest", ActionName: "directly_addressed"},
		{State: "pending", TargetType: "Issue", ActionName: "mentioned"},
		{State: "pending", TargetType: "Commit", ActionName: "mentioned"},
		{State: "pending", TargetType: "WikiPage::Meta", ActionName: "mentioned"},
		{State: "done", TargetType: "MergeRequest", ActionName: "assigned"},
	}}
	m.Todos[0].Target.IID = "412"
	m.Todos[1].Target.IID = "128"

	got := BuildIndex(m).Todos
	if len(got) != 2 {
		t.Fatalf("kept %d todos, want 2 (a commit, a wiki page and a done item are all out)", len(got))
	}
	for _, it := range got {
		if strings.HasSuffix(it.Ref, "0") && len(it.Ref) == 2 {
			t.Errorf("todo rendered as %q - the shell's !0 bug", it.Ref)
		}
	}
	if got[0].Ref != "!412" && got[1].Ref != "!412" {
		t.Errorf("merge request todo lost its ref: %+v", got)
	}
}

// A todo for a merge request that already has a band of its own must not appear twice;
// its reason belongs on that row.
func TestInboxFoldsTodoReasonIntoItsRow(t *testing.T) {
	t.Parallel()
	now := frozen(t)
	rows := BuildIndex(loadFixture(t)).Rows(ViewInbox, now)

	seen := map[string]int{}
	for _, r := range rows {
		seen[r.Ref]++
	}
	for ref, n := range seen {
		if n > 1 {
			t.Errorf("%s appears %d times in the inbox", ref, n)
		}
	}
	var found bool
	for _, r := range rows {
		if r.Ref == "!412" {
			found = true
			if !strings.Contains(r.Note, "directly addressed") {
				t.Errorf("!412 note = %q, want the todo reason folded in", r.Note)
			}
			if r.Label == todoBand {
				t.Errorf("!412 stayed in the todo band instead of its own")
			}
		}
	}
	if !found {
		t.Error("!412 is missing from the inbox entirely")
	}
}

// Ages are stored as epochs and formatted at read time. If they were baked into the
// index at sync time, every row would be a day wrong by morning - so the same index read
// a day later must age, and nothing else about it may move.
func TestAgesFollowTheClockNotTheSync(t *testing.T) {
	t.Parallel()
	idx := BuildIndex(loadFixture(t))
	now := frozen(t)

	today := idx.Rows(ViewMRs, now)
	tomorrow := idx.Rows(ViewMRs, now.AddDate(0, 0, 1))
	if len(today) != len(tomorrow) {
		t.Fatalf("row count changed with the clock: %d then %d", len(today), len(tomorrow))
	}

	aged := 0
	for i := range today {
		a, b := today[i], tomorrow[i]
		if a.Ref != b.Ref || a.Label != b.Label || a.Note != b.Note || a.Title != b.Title {
			t.Errorf("%s changed more than its age: %+v -> %+v", a.Ref, a, b)
		}
		if a.Age != b.Age {
			aged++
		}
	}
	if aged == 0 {
		t.Error("no row aged a day later; the age is baked in rather than computed")
	}
}
