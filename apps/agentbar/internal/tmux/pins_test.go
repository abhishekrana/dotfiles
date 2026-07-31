package tmux

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Pin names round-trip through @agentbar-pins even with spaces in them:
// tmux allows a space in a session name, so a space-separated list shredded
// "my repo" into two bogus pins and the row never read back as pinned.
func TestPinListRoundTripsNamesWithSpaces(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir()) // no mirror on disk to fall back to
	want := map[string]bool{"dotfiles": true, "my repo": true, "two  spaces": true}
	value := PinList(want)
	if got := ParsePins(value); len(got) != len(want) {
		t.Fatalf("ParsePins(%q) = %v, want %v", value, got, want)
	}
	r := &fakeRunner{replies: map[string]string{
		"show-option -gqv @agentbar-pins": value,
	}}
	got := Pins(r)
	for name := range want {
		if !got[name] {
			t.Errorf("pin %q lost in the round trip (value %q, got %v)", name, value, got)
		}
	}
	// An unset option is an empty set, not a set holding "".
	if empty := Pins(&fakeRunner{}); len(empty) != 0 {
		t.Errorf("Pins on unset option = %v, want empty", empty)
	}
}

// SetPins writes the disk mirror as well as the option, and Pins restores from
// it when the option is empty - a tmux server restart drops user options, and
// pins are the only thing that reorders the sidebar.
func TestPinsSurviveAServerRestart(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)

	if err := SetPins(&fakeRunner{}, map[string]bool{"dotfiles": true, "my repo": true}); err != nil {
		t.Fatalf("SetPins: %v", err)
	}
	if _, err := os.Stat(filepath.Join(state, "dotfiles", "agentbar-pins")); err != nil {
		t.Fatalf("mirror not written: %v", err)
	}

	// Fresh server: the option is unset, so Pins restores and stamps it back.
	fresh := &fakeRunner{}
	got := Pins(fresh)
	if !got["dotfiles"] || !got["my repo"] || len(got) != 2 {
		t.Errorf("Pins after restart = %v, want dotfiles + my repo", got)
	}
	if want := "set-option -g @agentbar-pins dotfiles\tmy repo"; !slices.Contains(fresh.calls, want) {
		t.Errorf("restore did not stamp the option back, calls = %v", fresh.calls)
	}

	// Unpinning everything must not resurrect the old set on the next read.
	if err := SetPins(&fakeRunner{}, map[string]bool{}); err != nil {
		t.Fatalf("SetPins(empty): %v", err)
	}
	if got := Pins(&fakeRunner{}); len(got) != 0 {
		t.Errorf("Pins after unpinning all = %v, want empty", got)
	}
}

// A pinned session that gets killed must not sit in the mirror forever now
// that the mirror outlives the server - the next pin write forgets it.
func TestSetPinsDropsDeadSessions(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	live := &fakeRunner{replies: map[string]string{
		"list-sessions -F #{session_name}": "dotfiles\napi",
	}}
	if err := SetPins(live, map[string]bool{"dotfiles": true, "killed-months-ago": true}); err != nil {
		t.Fatalf("SetPins: %v", err)
	}
	if got := Pins(&fakeRunner{}); len(got) != 1 || !got["dotfiles"] {
		t.Errorf("stored pins = %v, want just dotfiles", got)
	}
	// An unlistable server must not wipe the set.
	kept := &fakeRunner{}
	if err := SetPins(kept, map[string]bool{"dotfiles": true}); err != nil {
		t.Fatalf("SetPins: %v", err)
	}
	if got := Pins(&fakeRunner{}); !got["dotfiles"] {
		t.Errorf("pins = %v, want dotfiles kept when the session list is empty", got)
	}
}
