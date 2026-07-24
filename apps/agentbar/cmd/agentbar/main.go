// agentbar: a left tmux sidebar showing every Claude Code agent
// across all sessions and its state (working / needs attention / done).
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/abhishekrana/agentbar/internal/hook"
	"github.com/abhishekrana/agentbar/internal/tmux"
	"github.com/abhishekrana/agentbar/internal/trace"
	"github.com/abhishekrana/agentbar/internal/ui"
)

const usage = `usage: agentbar <command>

commands:
  run [--theme <name>]          run the live sidebar (inside a tmux pane)
  mockup [--theme <name>]       render the sidebar with fake data (visual preview)
  status                        print a status-line segment (⚠N ●N)
  hook                          Claude Code hook entry: stdin JSON -> pane options

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
		fmt.Print(tmux.StatusSegment(tmux.Exec{}))
	case "hook":
		runHook()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

// runHook never exits non-zero: a broken sidebar must not block Claude.
func runHook() {
	// Parse before the pane check so a paneless drop can log what the event
	// carried. Resumed/`claude daemon run` sessions fire hooks without
	// TMUX_PANE; recording their name/session/cwd here is the data a fallback
	// pane resolver will be built on.
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
	pane := os.Getenv("TMUX_PANE")
	if pane == "" {
		trace.Log("hook", "drop", "reason", "no_pane", "name", ev.Name,
			"sid", ev.SessionID, "cwd", ev.Cwd, "proj", os.Getenv("CLAUDE_PROJECT_DIR"))
		return
	}
	r := tmux.Exec{}
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
	// Ground truth for state-drift debugging: every event Claude sent us, the
	// state it moved the pane to, and whether the write failed. sid ties a line
	// to its session (so a resume/fork that swaps session id is visible), and
	// SessionStart's source (resume/fork/compact/…) flags how it began.
	fields := []any{"name", ev.Name, "prev", prev, "new", string(ef.State),
		"pane", pane, "sid", ev.SessionID}
	if ev.Source != "" {
		fields = append(fields, "source", ev.Source)
	}
	fields = append(fields, "err", trace.Err(applyErr))
	trace.Log("hook", "event", fields...)
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
