package model

import (
	"slices"
	"testing"
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
	in := []Session{
		{Name: "zeta", Agents: []Agent{{}}}, // active
		{Name: "alpha"},                     // dormant (no agents)
		{Name: "mid", Agents: []Agent{{}}},  // active, will be pinned
		{Name: "beta"},                      // dormant
		{Name: "yak", Agents: []Agent{{}}},  // active
	}
	out := Arrange(in, map[string]bool{"mid": true})

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
	got := Names(Arrange(sessions, map[string]bool{"blog": true, "dotfiles": true}))
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
