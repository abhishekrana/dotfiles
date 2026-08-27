package workdesk

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
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
			if r.Label == TodoBand {
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

// The model holds no presentation. A pre-padded title looks harmless until a UI sizes the
// column to the terminal and re-pads it: the trailing spaces get truncated and the row
// grows an ellipsis it never earned.
func TestIndexHoldsRawTitles(t *testing.T) {
	t.Parallel()
	idx := BuildIndex(loadFixture(t))
	check := func(kind, ref, title string) {
		if title != strings.TrimRight(title, " ") {
			t.Errorf("%s %s title is padded: %q", kind, ref, title)
		}
	}
	for _, it := range idx.MRs {
		check("mr", it.Ref, it.Title)
	}
	for _, it := range idx.Issues {
		check("issue", it.Ref, it.Title)
	}
	for _, it := range idx.Todos {
		check("todo", it.Ref, it.Title)
	}
	// The TSV is the one place a fixed column belongs, because its consumer is not this
	// program.
	rows := idx.Rows(ViewMRs, frozen(t))
	if len(rows) == 0 {
		t.Fatal("no rows")
	}
	field := strings.Split(rows[0].TSV(), "\t")[3]
	if n := len([]rune(field)); n != titleWidth {
		t.Errorf("TSV title field is %d runes, want %d", n, titleWidth)
	}
}

// The todo feed is not an inbox. GitLab never marks todos done, so the pending list is an
// accumulating log: on a real account it runs back years and is dominated by machine
// notifications about state the bands already derive. Measured, not assumed - of 453
// pending todos on one real account, 427 were review_submitted, build_failed, unmergeable
// or merge_train_removed, and not one was a mention.
func TestTodosKeepOnlyWhatTheBandsCannotDerive(t *testing.T) {
	t.Parallel()
	kept, dropped := buildTodos([]Todo{
		todo("assigned", "MergeRequest", "991"),           // not derivable: not your MR
		todo("mentioned", "Issue", "121"),                 // not derivable
		todo("directly_addressed", "MergeRequest", "412"), // not derivable
		todo("marked", "Issue", "77"),                     // you flagged it yourself
		todo("review_submitted", "MergeRequest", "377"),   // you reviewed someone else's
		todo("build_failed", "MergeRequest", "406"),       // the pipeline band says this
		todo("unmergeable", "MergeRequest", "368"),        // the gates say this
		todo("merge_train_removed", "MergeRequest", "402"),
		todo("mentioned", "Commit", "0"), // no view for a commit
	})

	if len(kept) != 4 {
		t.Errorf("kept %d todos, want 4", len(kept))
		for _, it := range kept {
			t.Logf("  kept %s %q", it.Ref, it.Note)
		}
	}
	if dropped != 5 {
		t.Errorf("dropped %d, want 5 - the count is shown, so it must be right", dropped)
	}
	for _, it := range kept {
		switch it.Note {
		case "assigned", "mentioned", "directly addressed", "marked":
		default:
			t.Errorf("kept a todo the bands already derive: %q", it.Note)
		}
	}
}

// A todo you have not acted on in a fortnight is a decision you already made. Aged out at
// read time, not at sync: a todo 13 days old when the snapshot was taken is 15 days old
// two days later, and the cutoff has to follow the clock rather than the snapshot.
func TestStaleTodosAgeOutAtReadTime(t *testing.T) {
	t.Parallel()
	now := frozen(t)
	m := &Mirror{Todos: []Todo{
		{ID: 1, State: "pending", ActionName: "assigned", TargetType: "MergeRequest",
			CreatedAt: now.Add(-2 * 24 * time.Hour).Format(time.RFC3339)},
		{ID: 2, State: "pending", ActionName: "assigned", TargetType: "MergeRequest",
			CreatedAt: now.Add(-200 * 24 * time.Hour).Format(time.RFC3339)},
	}}
	m.Todos[0].Target.IID = "1"
	m.Todos[1].Target.IID = "2"
	m.Todos[0].Target.Title = "recent"
	m.Todos[1].Target.Title = "ancient"

	idx := BuildIndex(m)
	if len(idx.Todos) != 2 {
		t.Fatalf("the index dropped a todo at build time: %d kept, want both", len(idx.Todos))
	}

	rows := idx.Rows(ViewInbox, now)
	if len(rows) != 1 {
		t.Fatalf("the inbox shows %d rows, want only the recent todo", len(rows))
	}
	if rows[0].Ref != "!1" {
		t.Errorf("kept %s, want the recent one", rows[0].Ref)
	}
	// And the same index read further into the future drops it too.
	if later := idx.Rows(ViewInbox, now.Add(30*24*time.Hour)); len(later) != 0 {
		t.Errorf("a month later the inbox still shows %d todos", len(later))
	}
}

func todo(action, target, iid string) Todo {
	td := Todo{
		State: "pending", ActionName: action, TargetType: target,
		CreatedAt: "2026-08-27T09:00:00Z",
	}
	td.Target.IID = flexID(iid)
	td.Target.Title = "a thing"
	return td
}

// Every view sorts the same way inside a band: newest first. Three of the six used to
// disagree, which opened a band with something seven months old.
func TestEveryViewSortsNewestFirst(t *testing.T) {
	t.Parallel()
	now := frozen(t)
	idx := BuildIndex(loadFixture(t))

	for _, c := range []struct {
		view View
		rows []Row
	}{
		{ViewMRs, idx.Rows(ViewMRs, now)},
		{ViewIssues, idx.Rows(ViewIssues, now)},
		{ViewInbox, idx.Rows(ViewInbox, now)},
	} {
		t.Run(c.view.String(), func(t *testing.T) {
			t.Parallel()
			epochs := map[string][]int64{}
			for _, r := range c.rows {
				epochs[r.Label] = append(epochs[r.Label], epochOf(idx, r.Ref))
			}
			for band, list := range epochs {
				for i := 1; i < len(list); i++ {
					if list[i-1] < list[i] {
						t.Errorf("%s band %q: row %d is older than the one after it",
							c.view, band, i)
					}
				}
			}
		})
	}
}

func epochOf(idx *Index, ref string) int64 {
	for _, it := range idx.MRs {
		if it.Ref == ref {
			return it.Updated
		}
	}
	for _, it := range idx.Issues {
		if it.Ref == ref {
			return it.Updated
		}
	}
	for _, it := range idx.Todos {
		if it.Ref == ref {
			return it.Updated
		}
	}
	return 0
}
