// Package tmux is a thin exec wrapper around the tmux CLI.
package tmux

import (
	"os/exec"
	"strings"
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
	out, err := exec.Command("tmux", args...).Output()
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
