// Package doctor is agentbar's self-check: it audits the tmux panes running
// Claude against the recent hook trace and flags the state-desync signatures -
// panes that never registered, sessions leaning on the cwd fallback (resumed /
// `claude daemon run`), and panes whose hooks are dropping without recovery so
// their sidebar state is stale. Parse and Render are pure; cmd/agentbar feeds
// them live tmux + trace output.
package doctor

import (
	"fmt"
	"strconv"
	"strings"
)

// Pane is one Claude pane's stamped agent state, from list-panes.
type Pane struct {
	Session, ID, State, Path string
	Present                  bool
	Since                    int64 // unix secs of the last state change (0 = unset)
}

// Health is the recent hook picture from the trace: paneless drops the cwd
// fallback could not place (keyed by cwd) and events it did recover (by pane).
type Health struct {
	NoPaneByCwd     map[string]int
	RecoveredByPane map[string]int
}

// PaneFormat is the tab-separated list-panes format ParsePanes expects.
const PaneFormat = "#{session_name}\t#{pane_id}\t#{pane_current_command}\t" +
	"#{pane_current_path}\t#{@agent_present}\t#{@agent_state}\t#{@agent_since}"

// ParsePanes keeps only Claude panes (command claude/node) from list-panes output.
func ParsePanes(out string) []Pane {
	var panes []Pane
	for ln := range strings.SplitSeq(strings.TrimRight(out, "\n"), "\n") {
		f := strings.Split(ln, "\t")
		if len(f) < 7 || (f[2] != "claude" && f[2] != "node") {
			continue
		}
		since, _ := strconv.ParseInt(f[6], 10, 64)
		panes = append(panes, Pane{
			Session: f[0], ID: f[1], Path: f[3], Present: f[4] == "1", State: f[5], Since: since,
		})
	}
	return panes
}

// ParseHealth reads `dotfiles-trace show --src hook` lines into a Health: a
// no_pane drop counts against its cwd, a via=cwd event against its pane.
func ParseHealth(traceOutput string) Health {
	h := Health{NoPaneByCwd: map[string]int{}, RecoveredByPane: map[string]int{}}
	for ln := range strings.SplitSeq(traceOutput, "\n") {
		switch {
		case strings.Contains(ln, "reason=no_pane"):
			h.NoPaneByCwd[fieldValue(ln, "cwd")]++
		case strings.Contains(ln, "via=cwd"):
			if p := fieldValue(ln, "pane"); p != "" {
				h.RecoveredByPane[p]++
			}
		}
	}
	return h
}

// fieldValue returns the logfmt value of " key=" in line, unquoting if needed.
func fieldValue(line, key string) string {
	i := strings.Index(line, " "+key+"=")
	if i < 0 {
		return ""
	}
	v := line[i+len(key)+2:]
	if strings.HasPrefix(v, `"`) {
		v = v[1:]
		var b strings.Builder
		for j := 0; j < len(v); j++ {
			switch {
			case v[j] == '\\' && j+1 < len(v):
				j++
				b.WriteByte(v[j])
			case v[j] == '"':
				return b.String()
			default:
				b.WriteByte(v[j])
			}
		}
		return b.String()
	}
	if sp := strings.IndexByte(v, ' '); sp >= 0 {
		v = v[:sp]
	}
	return v
}

// Render audits each live Claude pane by joining its drops (by cwd) and
// recoveries (by pane): a pane recovering via the fallback is healthy; one
// with recent drops and no recovery is stale; one that never registered is
// broken. now is unix secs, for the age column.
func Render(panes []Pane, h Health, now int64) string {
	var b strings.Builder
	b.WriteString("agentbar doctor\n\n")

	healthy, viaFallback, stale, notReg := 0, 0, 0, 0
	fmt.Fprintf(&b, "Claude panes (%d)\n", len(panes))
	for _, p := range panes {
		glyph, note := "✓", ""
		switch {
		case !p.Present:
			glyph, note, notReg = "✗", "not registered — no hook has landed", notReg+1
		case h.RecoveredByPane[p.ID] > 0:
			glyph = "ℹ"
			note = fmt.Sprintf("tracking via cwd fallback (%d) — resumed/daemon session", h.RecoveredByPane[p.ID])
			viaFallback++
		case h.NoPaneByCwd[p.Path] > 0:
			glyph = "⚠"
			note = fmt.Sprintf("%d hook(s) dropped, none recovered — state may be stale", h.NoPaneByCwd[p.Path])
			stale++
		default:
			healthy++
		}
		age := "-"
		if p.Since > 0 {
			age = humanAge(now - p.Since)
		}
		fmt.Fprintf(&b, "  %s %-13s %-5s %-8s %-5s %s\n",
			glyph, p.Session, p.ID, orDash(p.State), age, note)
	}

	fmt.Fprintf(&b, "\n%d healthy · %d via fallback · %d stale · %d not registered\n",
		healthy, viaFallback, stale, notReg)
	if stale > 0 || notReg > 0 {
		b.WriteString("Detail: dotfiles-trace show --src hook --grep no_pane\n")
	}
	return b.String()
}

// SidebarPane is one running sidebar and the flavor it was launched with.
type SidebarPane struct{ Session, ID, Theme string }

// SidebarFormat is the tab-separated list-panes format ParseSidebars expects.
const SidebarFormat = "#{session_name}\t#{pane_id}\t#{pane_current_command}\t#{pane_start_command}"

// Theme is one flavor as each of its three stores sees it.
type Theme struct {
	Configured string // ~/.config/theme/current - the persisted choice
	Option     string // @agentbar-theme - what the next sidebar will launch with
	Sidebars   []SidebarPane
}

// ParseSidebars keeps panes running the sidebar and reads back the --theme they
// were started with. Liveness is pane_current_command == agentbar, as elsewhere.
func ParseSidebars(out string) []SidebarPane {
	var panes []SidebarPane
	for ln := range strings.SplitSeq(strings.TrimRight(out, "\n"), "\n") {
		f := strings.Split(ln, "\t")
		if len(f) < 4 || f[2] != "agentbar" {
			continue
		}
		panes = append(panes, SidebarPane{Session: f[0], ID: f[1], Theme: themeFlag(f[3])})
	}
	return panes
}

// themeFlag returns the --theme value in a pane's start command, which tmux
// reports quoted.
func themeFlag(cmd string) string {
	_, after, found := strings.Cut(cmd, "--theme")
	if !found {
		return ""
	}
	v := strings.TrimLeft(after, " =")
	if j := strings.IndexAny(v, " \""); j >= 0 {
		v = v[:j]
	}
	return v
}

// RenderTheme reports flavor drift. A sidebar bakes --theme into its pane at
// spawn and reads the option only then, so the option reaches a running sidebar
// only on restart, and anything setting the option directly bypasses the file.
func RenderTheme(t Theme) string {
	var b strings.Builder
	b.WriteString("\nTheme\n")
	fmt.Fprintf(&b, "  configured  %-17s ~/.config/theme/current\n", orDash(t.Configured))
	fmt.Fprintf(&b, "  option      %-17s @agentbar-theme (the next sidebar's flavor)\n", orDash(t.Option))
	fmt.Fprintf(&b, "  sidebars    %s\n", tally(t.Sidebars))

	drift := false
	if t.Configured != "" && t.Option != "" && t.Configured != t.Option {
		drift = true
		fmt.Fprintf(&b, "  ✗ option ≠ configured — set outside `theme`; fix: theme %s\n", t.Configured)
	}
	stale := 0
	for _, s := range t.Sidebars {
		if t.Option != "" && s.Theme != t.Option {
			stale++
		}
	}
	if stale > 0 {
		drift = true
		fmt.Fprintf(&b, "  ⚠ %d sidebar(s) render an older flavor — restart: prefix + e twice\n", stale)
	}
	if !drift {
		b.WriteString("  ✓ in sync\n")
	}
	return b.String()
}

// tally counts sidebars per flavor, in first-seen order.
func tally(panes []SidebarPane) string {
	counts, order := map[string]int{}, []string{}
	for _, p := range panes {
		if counts[p.Theme] == 0 {
			order = append(order, p.Theme)
		}
		counts[p.Theme]++
	}
	if len(order) == 0 {
		return "none running"
	}
	parts := make([]string, 0, len(order))
	for _, th := range order {
		parts = append(parts, fmt.Sprintf("%d × %s", counts[th], orDash(th)))
	}
	return strings.Join(parts, ", ")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// humanAge renders a compact, non-negative age like 12s, 4m, 1h20m.
func humanAge(secs int64) string {
	if secs < 0 {
		secs = 0
	}
	switch {
	case secs < 60:
		return fmt.Sprintf("%ds", secs)
	case secs < 3600:
		return fmt.Sprintf("%dm", secs/60)
	default:
		return fmt.Sprintf("%dh%02dm", secs/3600, (secs%3600)/60)
	}
}
