package model

import (
	"slices"
	"testing"
	"time"
)

func names(sessions []Session) []string {
	out := make([]string, len(sessions))
	for i, s := range sessions {
		out[i] = s.Name
	}
	return out
}

// Arrange groups sessions into pinned / active / dormant bands, alphabetical
// within each, and stamps Pinned - without mutating the input.
func TestArrangeGroupsSortsAndStamps(t *testing.T) {
	// A bare Agent{} has no Since, so it is not fresh: give the active ones a
	// recent state change, which is what puts them in the active band.
	live := Agent{Since: time.Now().Add(-time.Minute)}
	in := []Session{
		{Name: "zeta", Agents: []Agent{live}}, // active
		{Name: "alpha"},                       // dormant (no agents)
		{Name: "mid", Agents: []Agent{live}},  // active, will be pinned
		{Name: "beta"},                        // dormant
		{Name: "yak", Agents: []Agent{live}},  // active
	}
	out := Arrange(in, map[string]bool{"mid": true}, time.Now(), time.Hour)

	want := []string{"mid", "yak", "zeta", "alpha", "beta"}
	if got := names(out); !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
	if !out[0].Pinned {
		t.Error("pinned session must have Pinned set")
	}
	for _, s := range out[1:] {
		if s.Pinned {
			t.Errorf("%s must not be pinned", s.Name)
		}
	}
	// Input slice is untouched (order and flags).
	if got := names(in); !slices.Equal(got, []string{"zeta", "alpha", "mid", "beta", "yak"}) {
		t.Errorf("Arrange mutated input order: %v", got)
	}
	if in[2].Pinned {
		t.Error("Arrange mutated the input's Pinned flag")
	}
}

func TestBand(t *testing.T) {
	cases := []struct {
		s    Session
		want int
	}{
		{Session{Pinned: true}, 0},
		{Session{Pinned: true, Agents: []Agent{{}}}, 0}, // pinned wins over agent count
		{Session{Agents: []Agent{{}}}, 1},
		{Session{}, 2}, // no agents
	}
	for _, c := range cases {
		if got := c.s.Band(); got != c.want {
			t.Errorf("Band(%+v) = %d, want %d", c.s, got, c.want)
		}
	}
}

func TestBandLabel(t *testing.T) {
	cases := []struct {
		s    Session
		want string
	}{
		{Session{Pinned: true}, "pinned"},
		{Session{Agents: []Agent{{}}}, "active"},
		{Session{}, "dormant"},
	}
	for _, c := range cases {
		if got := c.s.BandLabel(); got != c.want {
			t.Errorf("BandLabel(%+v) = %q, want %q", c.s, got, c.want)
		}
	}
}

// Step is what the session keys walk: the rendered order top to bottom and
// bottom to top, wrapping at both ends so no session is a dead end.
func TestStepWalksTheOrderAndWraps(t *testing.T) {
	names := []string{"blog", "dotfiles", "api", "payments"}
	cases := []struct {
		cur   string
		delta int
		want  string
	}{
		{"dotfiles", 1, "api"},   // down the bar, not alphabetically
		{"dotfiles", -1, "blog"}, // up
		{"payments", 1, "blog"},  // wrap past the bottom
		{"blog", -1, "payments"}, // wrap past the top
		{"gone", 1, "blog"},      // unknown session: enter from the top
		{"gone", -1, "payments"}, // ... and from the bottom going up
	}
	for _, c := range cases {
		if got := Step(names, c.cur, c.delta); got != c.want {
			t.Errorf("Step(%q, %d) = %q, want %q", c.cur, c.delta, got, c.want)
		}
	}
	if got := Step(nil, "dotfiles", 1); got != "" {
		t.Errorf("Step on an empty list = %q, want empty", got)
	}
	if got := Step([]string{"only"}, "only", 1); got != "only" {
		t.Errorf("Step on a single session = %q, want it to stay put", got)
	}
}

// Names must come off Arrange in band order - the keys walk this list, so an
// alphabetical slip here is exactly the jarring jump the bands removed.
func TestNamesFollowsTheBands(t *testing.T) {
	sessions := []Session{
		{Name: "payments"},                      // dormant
		{Name: "api", Agents: []Agent{{}}},      // active
		{Name: "blog", Agents: []Agent{{}}},     // pinned below
		{Name: "dotfiles", Agents: []Agent{{}}}, // pinned below
	}
	got := Names(Arrange(sessions, map[string]bool{"blog": true, "dotfiles": true}, time.Now(), time.Hour))
	want := []string{"blog", "dotfiles", "api", "payments"}
	if len(got) != len(want) {
		t.Fatalf("Names = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Names = %v, want %v", got, want)
		}
	}
}

// BranchOf names the worktree a session's agents share; when they do not share
// one, the first wins and "+N" says how many others are in there.
func TestBranchOf(t *testing.T) {
	one := []Agent{{Branch: "main"}, {Branch: "main"}}
	if got := BranchOf(one); got != "main" {
		t.Errorf("agreed branch = %q, want main", got)
	}
	if got := BranchOf(nil); got != "" {
		t.Errorf("no agents = %q, want empty", got)
	}
	// An agent with no branch (a pane outside any repo) must not win the slot.
	if got := BranchOf([]Agent{{Branch: ""}, {Branch: "main"}}); got != "main" {
		t.Errorf("blank branch should not win, got %q", got)
	}
	two := []Agent{{Branch: "main"}, {Branch: "feat/x"}, {Branch: "feat/x"}}
	if got := BranchOf(two); got != "main +1" {
		t.Errorf("divergent branches = %q, want \"main +1\"", got)
	}
	three := []Agent{{Branch: "main"}, {Branch: "a"}, {Branch: "b"}}
	if got := BranchOf(three); got != "main +2" {
		t.Errorf("two others = %q, want \"main +2\"", got)
	}
}

// Fresh is what decides the active band: a live agent counts however long it has
// been at it, because @agent_since is the time of the last state *change* - a
// ninety-minute turn would otherwise age out mid-work.
func TestFresh(t *testing.T) {
	now := time.Now()
	old := now.Add(-3 * time.Hour)
	cases := map[string]struct {
		agents []Agent
		want   bool
	}{
		"working, long past its last change": {[]Agent{{State: StateWorking, Since: old}}, true},
		"blocked on you, however old":        {[]Agent{{State: StatePermission, Since: old}}, true},
		"asking, however old":                {[]Agent{{State: StateQuestion, Since: old}}, true},
		"done a moment ago":                  {[]Agent{{State: StateDone, Since: now.Add(-time.Minute)}}, true},
		"done three hours ago":               {[]Agent{{State: StateDone, Since: old}}, false},
		"idle three hours ago":               {[]Agent{{State: StateIdle, Since: old}}, false},
		"no timestamp at all":                {[]Agent{{State: StateDone}}, false},
		"one cold, one warm":                 {[]Agent{{State: StateDone, Since: old}, {State: StateDone, Since: now}}, true},
		"no agents":                          {nil, false},
	}
	for name, c := range cases {
		if got := Fresh(c.agents, now, time.Hour); got != c.want {
			t.Errorf("%s: Fresh = %v, want %v", name, got, c.want)
		}
	}
}

// A session whose agents have all gone quiet sinks to dormant on the clock
// alone - but a pinned one never moves, because pins are the user's.
func TestArrangeSinksQuietSessionsButNeverPinned(t *testing.T) {
	now := time.Now()
	cold := Agent{State: StateDone, Since: now.Add(-3 * time.Hour)}
	warm := Agent{State: StateDone, Since: now.Add(-time.Minute)}
	in := []Session{
		{Name: "cold", Agents: []Agent{cold}},
		{Name: "warm", Agents: []Agent{warm}},
		{Name: "pinned-cold", Agents: []Agent{cold}},
	}
	out := Arrange(in, map[string]bool{"pinned-cold": true}, now, time.Hour)

	byName := map[string]Session{}
	for _, s := range out {
		byName[s.Name] = s
	}
	if got := byName["cold"].Band(); got != 2 {
		t.Errorf("a quiet session should sink to dormant, got band %d", got)
	}
	if !byName["cold"].Quiet {
		t.Error("a quiet session should be stamped Quiet")
	}
	if got := byName["warm"].Band(); got != 1 {
		t.Errorf("a session with recent activity stays active, got band %d", got)
	}
	if got := byName["pinned-cold"].Band(); got != 0 {
		t.Errorf("a pinned session must not move, however quiet: got band %d", got)
	}
	// The window is what decides it: widen it and the cold one comes back.
	wide := Arrange(in, nil, now, 4*time.Hour)
	for _, s := range wide {
		if s.Name == "cold" && s.Band() != 1 {
			t.Errorf("with a 4h window the cold session is active, got band %d", s.Band())
		}
	}
	// An unset (zero) window must not flatten the bar to dormant.
	for _, s := range Arrange(in, nil, now, 0) {
		if s.Name == "warm" && s.Band() != 1 {
			t.Error("a zero window should fall back to the default, not sink everything")
		}
	}
}
