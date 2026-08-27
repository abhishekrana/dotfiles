// Package picker drives fzf.
//
// fzf only ever picks here: --expect hands the pressed key back and the caller
// dispatches it, so every action stays an ordinary function that can be run without a
// terminal. Nothing is hidden inside an fzf execute binding.
package picker

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// BandMark is the first field of a row that is a band header rather than an item. The
// cursor steps over these and Enter ignores them.
const BandMark = "__band__"

// ErrClosed means the person dismissed the picker without choosing.
var ErrClosed = errors.New("picker closed")

// Options is one fzf invocation, as data. Building the argument list from a struct
// rather than a string is what keeps quoting out of it.
type Options struct {
	// Rows are tab-separated; the first field is the reference an action acts on and
	// the second is what the eye reads.
	Rows []string
	// Header is shown above the list. Keep every line under about 52 columns: the list
	// is under half the popup and fzf truncates a long header silently, taking the
	// last keys with it.
	Header string
	// Preview is the command fzf runs for the highlighted row, with {1} standing for
	// its first field.
	Preview string
	// Keys fzf should hand back instead of acting on itself.
	Keys []string
	// Colors is the shared palette string, empty before the first theme run.
	Colors string
}

// Result is what the person did: the key they pressed, and the row that was highlighted.
type Result struct {
	Key string
	Ref string
}

// Args is the fzf argument list.
//
// The band-skipping bindings are the one place a shell fragment survives into Go, and
// they have to: transform() is evaluated by fzf's own shell, so this is fzf's API rather
// than ours. The parenthesised form is required - `transform:` without parentheses
// swallows the rest of the --bind string and silently eats every binding after it.
func (o Options) Args() []string {
	skipDown := fmt.Sprintf("transform([ {1} = %s ] && echo down || true)", BandMark)
	skipUp := fmt.Sprintf(
		`transform([ {1} = %s ] && { [ "$FZF_POS" -gt 1 ] && echo up || echo down; })`, BandMark)

	args := []string{
		"--ansi", "--sync", "--reverse", "--no-input", "--highlight-line", "--no-multi",
		// Explicitly, to override the FZF_DEFAULT_OPTS inherited from the server
		// environment: its own --height leaves dead space and its --border draws a
		// second frame inside tmux's.
		"--height=100%", "--border=none",
		"--header=" + o.Header,
		"--header-first",
		"--delimiter=\t", "--with-nth=2", "--pointer= ",
		"--preview", o.Preview,
		"--preview-window=right,52%,border-left",
	}
	if o.Colors != "" {
		args = append(args, "--color="+o.Colors)
	}
	if len(o.Keys) > 0 {
		args = append(args, "--expect="+strings.Join(o.Keys, ","))
	}
	return append(args,
		"--bind", "start:pos(2)",
		"--bind", "j:down+"+skipDown+"+"+skipDown+",k:up+"+skipUp+"+"+skipUp,
		"--bind", "g:first+"+skipDown+"+"+skipDown+",G:last+"+skipUp+"+"+skipUp,
		"--bind", "/:toggle-input",
		"--bind", fmt.Sprintf("enter:transform([ {1} = %s ] && echo ignore || echo accept)", BandMark),
	)
}

// Run shows the picker and returns what was chosen.
func Run(o Options) (Result, error) {
	cmd := exec.Command("fzf", o.Args()...)
	cmd.Stdin = strings.NewReader(strings.Join(o.Rows, "\n") + "\n")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		// 130 is an interrupt and 1 is "nothing matched"; both mean the person is done.
		if errors.As(err, &exit) && (exit.ExitCode() == 130 || exit.ExitCode() == 1) {
			return Result{}, ErrClosed
		}
		return Result{}, fmt.Errorf("fzf: %w", err)
	}
	return parse(string(out)), nil
}

// parse reads fzf's two lines: the expected key, then the selected row.
func parse(out string) Result {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	var r Result
	if len(lines) > 0 {
		r.Key = strings.TrimSpace(lines[0])
	}
	if len(lines) > 1 {
		if ref, _, _ := strings.Cut(lines[1], "\t"); ref != BandMark {
			r.Ref = ref
		}
	}
	return r
}
