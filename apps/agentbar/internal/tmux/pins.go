package tmux

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RefreshChannel is the wait-for channel every sidebar blocks on between
// ticks; signalling it makes them all redraw now.
const RefreshChannel = "agentbar-refresh"

// pinsOption holds the live pinned set, tab-separated. Tab is the one
// separator a session name can never hold: tmux takes spaces (the picker's
// `c`/`r` prompts are free text) but rejects tabs.
const pinsOption = "@agentbar-pins"

// pinsFile mirrors the shell idiom: $XDG_STATE_HOME, else ~/.local/state.
// Resolved per call so a test can redirect it.
func pinsFile() string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".local", "state")
	}
	return filepath.Join(base, "dotfiles", "agentbar-pins")
}

// Pins reads the pinned-session set. An empty option means a tmux server that
// just started - user options die with it - so restore the disk mirror and
// stamp it back, keeping the bands you left behind across a restart. Pins are
// the only thing that reorders the sidebar, so losing them flattens it.
func Pins(r Runner) map[string]bool {
	out, _ := r.Run("show-option", "-gqv", pinsOption)
	if out == "" {
		if saved := readPinsFile(); saved != "" {
			_, _ = r.Run("set-option", "-g", pinsOption, saved)
			out = saved
		}
	}
	return ParsePins(out)
}

// ParsePins splits a stored value into a set, dropping empty fields.
func ParsePins(value string) map[string]bool {
	pins := map[string]bool{}
	for name := range strings.SplitSeq(value, "\t") {
		if name != "" {
			pins[name] = true
		}
	}
	return pins
}

// PinList serializes a set back to the sorted, tab-separated stored form.
func PinList(pins map[string]bool) string {
	names := make([]string, 0, len(pins))
	for name := range pins {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, "\t")
}

// SetPins persists a changed set - option, disk mirror, then a refresh signal
// so every sidebar regroups at once. Both writes always run, so the mirror
// cannot drift from the option.
func SetPins(r Runner, pins map[string]bool) error {
	value := PinList(prunePins(r, pins))
	_, err := r.Run("set-option", "-g", pinsOption, value, ";", "wait-for", "-S", RefreshChannel)
	if fileErr := writePinsFile(value); err == nil {
		err = fileErr
	}
	return err
}

// prunePins drops names no live session answers to. Killing a pinned session
// used to leave its name in the option until the server exited; the mirror
// outlives that, so a write is the moment to forget it. An unreadable server
// leaves the set alone - a stale name beats dropping every pin.
func prunePins(r Runner, pins map[string]bool) map[string]bool {
	out, err := r.Run("list-sessions", "-F", "#{session_name}")
	if err != nil || out == "" {
		return pins
	}
	live := map[string]bool{}
	for name := range strings.SplitSeq(out, "\n") {
		live[name] = true
	}
	kept := map[string]bool{}
	for name := range pins {
		if live[name] {
			kept[name] = true
		}
	}
	return kept
}

func readPinsFile() string {
	b, err := os.ReadFile(pinsFile())
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(b), "\n")
}

// writePinsFile replaces the mirror atomically: a torn write would lose pins.
func writePinsFile(value string) error {
	path := pinsFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".agentbar-pins-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once the rename below lands
	if _, err := tmp.WriteString(value + "\n"); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
