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

// The two persisted band sets, tab-separated. Tab is the one separator a
// session name can never hold: tmux takes spaces (the picker's `c`/`r` prompts
// are free text) but rejects tabs.
const (
	pinsOption = "@agentbar-pins"
	bandOption = "@agentbar-bands"
)

// stateFile mirrors the shell idiom: $XDG_STATE_HOME, else ~/.local/state.
// Resolved per call so a test can redirect it.
func stateFile(name string) string {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".local", "state")
	}
	return filepath.Join(base, "dotfiles", name)
}

// Pins reads the pinned-session set. An empty option means a tmux server that
// just started - user options die with it - so restore the disk mirror and
// stamp it back, keeping the bands you left behind across a restart. Pins are
// the only thing that reorders the sidebar, so losing them flattens it.
func Pins(r Runner) map[string]bool {
	return ParsePins(restore(r, pinsOption, "agentbar-pins"))
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
	return store(r, pinsOption, "agentbar-pins", PinList(prunePins(r, pins)))
}

// Bands reads the band a session was put in by hand: "active" or "dormant",
// from the `a` and `d` keys. Absent means the clock decides, which is the
// normal case - these are the exceptions you asked for by name.
func Bands(r Runner) map[string]string {
	return ParseBands(restore(r, bandOption, "agentbar-bands"))
}

// ParseBands splits the stored "name=band" records, dropping anything
// malformed or unknown rather than failing: one bad record must not cost the
// rest of the set.
func ParseBands(value string) map[string]string {
	out := map[string]string{}
	for rec := range strings.SplitSeq(value, "\t") {
		name, band, ok := strings.Cut(rec, "=")
		if !ok || name == "" {
			continue
		}
		switch band {
		case "active", "dormant":
			out[name] = band
		}
	}
	return out
}

// BandList serializes back to the sorted, tab-separated stored form.
func BandList(bands map[string]string) string {
	recs := make([]string, 0, len(bands))
	for name, band := range bands {
		recs = append(recs, name+"="+band)
	}
	sort.Strings(recs)
	return strings.Join(recs, "\t")
}

// SetBands persists a changed set, the same way pins are.
func SetBands(r Runner, bands map[string]string) error {
	live := prunePins(r, boolSet(bands))
	kept := map[string]string{}
	for name, band := range bands {
		if live[name] {
			kept[name] = band
		}
	}
	return store(r, bandOption, "agentbar-bands", BandList(kept))
}

func boolSet(bands map[string]string) map[string]bool {
	out := map[string]bool{}
	for name := range bands {
		out[name] = true
	}
	return out
}

// restore reads an option, falling back to its disk mirror and stamping it
// back: an empty option means a tmux server that just started, since user
// options die with it, and the bands you left behind should survive that.
func restore(r Runner, option, file string) string {
	out, _ := r.Run("show-option", "-gqv", option)
	if out == "" {
		if saved := readMirror(file); saved != "" {
			_, _ = r.Run("set-option", "-g", option, saved)
			out = saved
		}
	}
	return out
}

// store writes the option and the mirror, then signals every sidebar to
// regroup now. Both writes always run, so the two cannot drift.
func store(r Runner, option, file, value string) error {
	_, err := r.Run("set-option", "-g", option, value, ";", "wait-for", "-S", RefreshChannel)
	if fileErr := writeMirror(file, value); err == nil {
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

func readMirror(file string) string {
	b, err := os.ReadFile(stateFile(file))
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(b), "\n")
}

// writeMirror replaces a mirror atomically: a torn write would lose the set.
func writeMirror(file, value string) error {
	path := stateFile(file)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+file+"-*")
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
