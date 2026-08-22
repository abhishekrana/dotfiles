// agentbar: a left tmux sidebar showing every Claude Code agent
// across all sessions and its state (working / needs attention / done).
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/abhishekrana/agentbar/internal/doctor"
	"github.com/abhishekrana/agentbar/internal/hook"
	"github.com/abhishekrana/agentbar/internal/model"
	"github.com/abhishekrana/agentbar/internal/tmux"
	"github.com/abhishekrana/agentbar/internal/trace"
	"github.com/abhishekrana/agentbar/internal/ui"
)

const usage = `usage: agentbar <command>

commands:
  run [--theme <name>]          run the live sidebar (inside a tmux pane)
  mockup [--theme <name>]       render the sidebar with fake data (visual preview)
  status                        print a status-line segment (⚠N ●N)
  order                         print the sidebar's session order, "band<TAB>name" per line
  next | prev [<session> [<tty>]]
                                switch the client one session down / up that order
  band <session> pinned|active|dormant
                                put a session in a band by hand (the p/a/d keys)
  hook                          Claude Code hook entry: stdin JSON -> pane options
  doctor                        audit Claude panes vs the hook trace for state desync

themes: solarized-light (default), solarized-dark, catppuccin-latte, catppuccin-mocha
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run":
		runLive(os.Args[2:])
	case "mockup":
		runMockup(os.Args[2:])
	case "status":
		// The status segment speaks the same state language as the sidebar, so it
		// takes its colours from the configured flavor too.
		t := ui.ThemeByName(configuredTheme())
		fmt.Print(tmux.StatusSegment(tmux.Exec{}, string(t.Blocked), string(t.Working)))
	case "order":
		runOrder()
	case "next":
		runStep(1, os.Args[2:])
	case "prev":
		runStep(-1, os.Args[2:])
	case "band":
		runBand(os.Args[2:])
	case "hook":
		runHook()
	case "doctor":
		runDoctor()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

// ordered returns every session in sidebar order - the same model.Arrange
// bands the TUI renders, so the keys and the picker walk exactly what you see.
// A nil branch cache skips the per-pane git lookups: order needs no branches,
// and these run on a keypress.
func ordered(r tmux.Runner, current string) []model.Session {
	snap := tmux.Snapshot(r, nil, current)
	return model.Arrange(snap.Sessions, model.Grouping{
		Pinned:    tmux.Pins(r),
		Forced:    tmux.Bands(r),
		Now:       time.Now(),
		ActiveFor: tmux.ActiveFor(r),
	})
}

// runOrder publishes the order as "band<TAB>name" lines, the picker popup's
// source for grouping its rows into the same bands as the bar. Display (state
// glyphs, branch) stays the picker's own business; only the order is shared.
func runOrder() {
	r := tmux.Exec{}
	var b strings.Builder
	for _, s := range ordered(r, tmux.CurrentSession(r)) {
		b.WriteString(s.BandLabel() + "\t" + s.Name + "\n")
	}
	fmt.Print(b.String())
}

// runStep switches the client one row down (+1) or up (-1) the sidebar's
// order, wrapping at both ends. A non-zero exit lets the tmux binding fall
// back to switch-client, so the key still works without a built binary.
//
// args is the pressing client's own [session, tty]: the key bindings pass
// tmux's #{client_session} and #{client_tty}, so the walk starts at the right
// row and moves the right client even with several attached. Both fall back to
// a guess for a bare run from a shell. tmux does NOT re-stamp TMUX_PANE for a
// key binding's run-shell child - it inherits the server's environment - so a
// guessed session silently walks from the wrong row.
func runStep(delta int, args []string) {
	r := tmux.Exec{}
	cur, tty := "", ""
	if len(args) > 0 {
		cur = strings.TrimSpace(args[0])
	}
	if len(args) > 1 {
		tty = strings.TrimSpace(args[1])
	}
	if cur == "" {
		cur = tmux.CurrentSession(r)
	}
	target := model.Step(model.Names(ordered(r, cur)), cur, delta)
	if target == "" || target == cur {
		return // nothing to move to: a single session, or an empty server
	}
	cmd := []string{"switch-client"}
	if tty == "" {
		tty = tmux.ClientFor(r, cur)
	}
	if tty != "" {
		cmd = append(cmd, "-c", tty)
	}
	// Publishing the session (token "=name") is the same write the sidebar's
	// own Enter makes: every sidebar moves its highlight now, not next tick.
	cmd = append(cmd,
		"-t", target, ";",
		"set-option", "-g", "@sidebar_selected", "="+target, ";",
		"wait-for", "-S", tmux.RefreshChannel,
	)
	start := time.Now()
	_, err := r.Run(cmd...)
	trace.Log("agentbar", "switch", "session", target, "from", cur, "key", stepLabel(delta),
		"ms", time.Since(start).Milliseconds(), "err", trace.Err(err))
	if err != nil {
		fmt.Fprintln(os.Stderr, "agentbar:", err)
		os.Exit(1)
	}
}

func stepLabel(delta int) string {
	if delta < 0 {
		return "prev"
	}
	return "next"
}

// runBand puts one session in a band by hand - the sidebar's p, a and d keys as
// a command, so the picker drives the same two stores. One key, one
// destination; naming the band a session is already in changes nothing.
func runBand(args []string) {
	if len(args) != 2 || args[0] == "" {
		fmt.Fprint(os.Stderr, "usage: agentbar band <session> pinned|active|dormant\n")
		os.Exit(2)
	}
	name, want := args[0], args[1]
	switch want {
	case model.BandPinned, model.BandActive, model.BandDormant:
	default:
		fmt.Fprintf(os.Stderr, "agentbar: unknown band %q\n", want)
		os.Exit(2)
	}
	r := tmux.Exec{}
	pins, bands := model.Place(tmux.Pins(r), tmux.Bands(r), name, want)
	// Same one-shot rule the sidebar's `d` follows: a session being worked in
	// cannot be sunk, so the placement never lands. Both views share the store,
	// so both have to agree on what a keypress did.
	if next, expired := model.Expire(bands, tmux.Snapshot(r, nil, "").Sessions, name); expired {
		bands, want = next, "auto"
	}
	err := tmux.SetPins(r, pins)
	if bandErr := tmux.SetBands(r, bands); err == nil {
		err = bandErr
	}
	trace.Log("agentbar", "band", "session", name, "band", want, "err", trace.Err(err))
	if err != nil {
		fmt.Fprintln(os.Stderr, "agentbar:", err)
		os.Exit(1)
	}
}

// runHook never exits non-zero: a broken sidebar must not block Claude.
func runHook() {
	// Parse before the pane check so a paneless hook can be recovered (and, if
	// not, dropped with detail). Resumed/`claude daemon run` sessions fire
	// hooks without TMUX_PANE; fall back to matching the event's cwd to a
	// Claude pane so their state still tracks.
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		trace.Log("hook", "drop", "reason", "read_err", "err", trace.Err(err))
		return
	}
	var ev hook.Event
	if err := json.Unmarshal(data, &ev); err != nil {
		trace.Log("hook", "drop", "reason", "json_err", "err", trace.Err(err))
		return
	}
	r := tmux.Exec{}
	pane, via := os.Getenv("TMUX_PANE"), "env"
	if pane == "" {
		pane, via = hook.ResolvePane(r, ev)
	}
	if pane == "" {
		trace.Log("hook", "drop", "reason", "no_pane", "name", ev.Name,
			"sid", ev.SessionID, "cwd", ev.Cwd, "proj", os.Getenv("CLAUDE_PROJECT_DIR"))
		return
	}
	ef := hook.Decide(ev)
	prev := tmux.PaneOption(r, pane, "@agent_state") // state before Apply, for transition detection
	applyErr := hook.Apply(r, pane, ev, ef, time.Now())
	if applyErr != nil {
		fmt.Fprintln(os.Stderr, "agentbar:", applyErr)
	}
	notifyOpt, _ := r.Run("show-options", "-gqv", "@agent_notify")
	if hook.ShouldNotify(prev, ef, notifyOpt) {
		hook.Notify(r, pane, ef.State)
	}
	// A new or ended agent context must not inherit the previous session's write
	// target: the pane option outlives the Claude session. Before the stamp below,
	// so the two never fight over one event.
	if ef.ClearWorkdir {
		if wdFrom := hook.ClearWorkdir(r, pane); wdFrom != "" {
			trace.Log("hook", "workdir", "pane", pane, "before", wdFrom, "after", "")
		}
	}
	// Where the agent is writing, which the pane's cwd does not follow. Only a
	// change is traced or acted on; an edit in the same worktree costs nothing.
	if wdFrom, wdTo := hook.ApplyWorkdir(r, pane, ev); wdTo != "" {
		trace.Log("hook", "workdir", "pane", pane, "before", wdFrom, "after", wdTo)
		hook.RunWorkdirCmd(r, pane, wdTo)
	}
	// Ground truth for state-drift debugging: every event Claude sent us, the
	// state it moved the pane to, and whether the write failed. sid ties a line
	// to its session (so a resume/fork that swaps session id is visible), and
	// SessionStart's source (resume/fork/compact/…) flags how it began.
	fields := []any{"name", ev.Name, "prev", prev, "new", string(ef.State),
		"pane", pane, "sid", ev.SessionID}
	if via != "env" { // pane recovered via the cwd fallback, not TMUX_PANE
		fields = append(fields, "via", via)
	}
	if ev.Source != "" {
		fields = append(fields, "source", ev.Source)
	}
	fields = append(fields, "err", trace.Err(applyErr))
	trace.Log("hook", "event", fields...)
}

// runDoctor prints a one-shot health audit: every Claude pane's stamped state
// cross-referenced against recent hook drops/recoveries in the trace, so a
// stale sidebar (paneless hooks from resumed/daemon sessions) is one command
// to spot instead of an investigation.
func runDoctor() {
	panesOut, _ := tmux.Exec{}.Run("list-panes", "-a", "-F", doctor.PaneFormat)
	fmt.Print(doctor.Render(
		doctor.ParsePanes(panesOut),
		doctor.ParseHealth(traceHook("1h")),
		time.Now().Unix(),
	))

	sidebarsOut, _ := tmux.Exec{}.Run("list-panes", "-a", "-F", doctor.SidebarFormat)
	option, _ := tmux.Exec{}.Run("show-option", "-gqv", "@agentbar-theme")
	fmt.Print(doctor.RenderTheme(doctor.Theme{
		Configured: configuredTheme(),
		Option:     option,
		Sidebars:   doctor.ParseSidebars(sidebarsOut),
	}))
}

// configuredTheme reads the flavor the `theme` switcher persisted.
func configuredTheme() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	b, err := os.ReadFile(filepath.Join(dir, "theme", "current"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// traceHook returns recent `src=hook` lines via the dotfiles-trace CLI (the
// trace log's query tool); empty if it isn't reachable, so doctor degrades to
// the pane audit alone.
func traceHook(since string) string {
	bin := "dotfiles-trace"
	if _, err := exec.LookPath(bin); err != nil {
		bin = os.Getenv("HOME") + "/.local/bin/dotfiles-trace"
	}
	out, _ := exec.Command(bin, "show", "--src", "hook", "--since", since).Output()
	return string(out)
}

func themeFlag(args []string) ui.Theme {
	for i, a := range args {
		if a == "--theme" && i+1 < len(args) {
			return ui.ThemeByName(args[i+1])
		}
	}
	return ui.SolarizedLight()
}

func runMockup(args []string) {
	runTUI(ui.NewMockup(themeFlag(args)))
}

func runLive(args []string) {
	if os.Getenv("TMUX") == "" {
		fmt.Fprintln(os.Stderr, "error: run must be started inside a tmux pane")
		os.Exit(1)
	}
	runTUI(ui.NewLive(themeFlag(args)))
}

func runTUI(app ui.App) {
	if _, err := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseAllMotion()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
