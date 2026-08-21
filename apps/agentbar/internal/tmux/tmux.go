// Package tmux is a thin exec wrapper around the tmux CLI.
package tmux

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/abhishekrana/agentbar/internal/model"
)

// Runner abstracts tmux invocation so hook logic is testable.
type Runner interface {
	// Run executes one tmux invocation (args may contain ";" separators
	// for multiple tmux commands) and returns trimmed stdout.
	Run(args ...string) (string, error)
}

// Exec is the real Runner.
type Exec struct{}

func (Exec) Run(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	// tmux replaces tabs in -F output with "_" outside a UTF-8 locale, which
	// would shred every field this package splits on.
	cmd.Env = append(os.Environ(), "LC_ALL=C.UTF-8")
	out, err := cmd.Output()
	// Trim only newlines: a TrimSpace would eat trailing tabs of the
	// last output line, i.e. trailing empty format fields.
	return strings.TrimRight(string(out), "\n"), err
}

// PaneOption reads a pane-scoped user option; empty string if unset.
func PaneOption(r Runner, pane, name string) string {
	out, err := r.Run("show-options", "-pqv", "-t", pane, name)
	if err != nil {
		return ""
	}
	return out
}

// ActiveFor is how long a session stays in the active band after its last agent
// activity (@agentbar-active-for, e.g. "30m", "1h", "4h"). Unset or unparseable
// means model.DefaultActiveFor, so a fat-fingered option cannot flatten the bar.
func ActiveFor(r Runner) time.Duration {
	v, err := r.Run("show-option", "-gqv", "@agentbar-active-for")
	if err != nil {
		return model.DefaultActiveFor
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil || d <= 0 {
		return model.DefaultActiveFor
	}
	return d
}
