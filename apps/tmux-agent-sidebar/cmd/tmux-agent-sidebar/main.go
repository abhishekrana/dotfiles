// tmux-agent-sidebar: a left tmux sidebar showing every Claude Code agent
// across all sessions and its state (working / needs attention / done).
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/abhishekrana/tmux-agent-sidebar/internal/hook"
	"github.com/abhishekrana/tmux-agent-sidebar/internal/tmux"
	"github.com/abhishekrana/tmux-agent-sidebar/internal/ui"
)

const usage = `usage: tmux-agent-sidebar <command>

commands:
  run [--theme <name>]          run the live sidebar (inside a tmux pane)
  mockup [--theme <name>]       render the sidebar with fake data (visual preview)
  status                        print a status-line segment (⚠N ●N)
  hook                          Claude Code hook entry: stdin JSON -> pane options
  install-hooks [--target f]    add hook entries to Claude settings (default:
                                ~/.claude/settings.json); idempotent

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
	case "install-hooks":
		runInstallHooks(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

// runHook never exits non-zero: a broken sidebar must not block Claude.
func runHook() {
	pane := os.Getenv("TMUX_PANE")
	if pane == "" {
		return // agent not running inside tmux
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return
	}
	var ev hook.Event
	if err := json.Unmarshal(data, &ev); err != nil {
		return
	}
	r := tmux.Exec{}
	ef := hook.Decide(ev)
	prev := tmux.PaneOption(r, pane, "@agent_state") // state before Apply, for transition detection
	if err := hook.Apply(r, pane, ev, ef, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, "tmux-agent-sidebar:", err)
	}
	notifyOpt, _ := r.Run("show-options", "-gqv", "@agent_notify")
	if hook.ShouldNotify(prev, ef, notifyOpt) {
		hook.Notify(r, pane, ef.State)
	}
}

func runInstallHooks(args []string) {
	target := hook.DefaultSettingsPath()
	for i, a := range args {
		if a == "--target" && i+1 < len(args) {
			target = args[i+1]
		}
	}
	bin, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	changed, err := hook.Install(target, bin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if changed {
		fmt.Println("hooks installed in", target)
	} else {
		fmt.Println("hooks already installed in", target)
	}
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
