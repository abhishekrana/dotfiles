package doctor

import (
	"strings"
	"testing"
)

func TestParsePanesFiltersAndParses(t *testing.T) {
	out := "api\t%3\tclaude\t/ws/api\t1\tworking\t1700000000\n" +
		"api\t%4\tbash\t/ws/api\t\t\t\n" + // not an agent -> dropped
		"blog\t%9\tnode\t/ws/blog\t\tdone\t1699999000\n"
	panes := ParsePanes(out)
	if len(panes) != 2 {
		t.Fatalf("want 2 claude/node panes, got %d: %+v", len(panes), panes)
	}
	if panes[0].ID != "%3" || !panes[0].Present || panes[0].State != "working" || panes[0].Since != 1700000000 {
		t.Errorf("pane[0] parsed wrong: %+v", panes[0])
	}
	if panes[1].Present { // @agent_present empty -> not registered
		t.Errorf("node pane with empty @agent_present must be not-registered: %+v", panes[1])
	}
}

func TestParseHealthCountsDropsAndRecoveries(t *testing.T) {
	tr := strings.Join([]string{
		`ts=x src=hook evt=drop reason=no_pane name=PreToolUse sid=x cwd=/ws/api proj=/ws/api`,
		`ts=x src=hook evt=drop reason=no_pane name=Stop sid=x cwd=/ws/api proj=/ws/api`,
		`ts=x src=hook evt=event name=PreToolUse pane=%3 sid=x via=cwd err=""`,
		`ts=x src=hook evt=event name=Stop pane=%32 err=""`, // normal landing, not counted
	}, "\n")
	h := ParseHealth(tr)
	if h.NoPaneByCwd["/ws/api"] != 2 {
		t.Errorf("want 2 drops for /ws/api, got %d", h.NoPaneByCwd["/ws/api"])
	}
	if h.RecoveredByPane["%3"] != 1 {
		t.Errorf("want 1 recovery for %%3, got %d", h.RecoveredByPane["%3"])
	}
}

func TestFieldValue(t *testing.T) {
	line := `ts=x src=hook cwd=/ws/api name="Ask User" pane=%3`
	cases := map[string]string{"cwd": "/ws/api", "name": "Ask User", "pane": "%3", "missing": ""}
	for key, want := range cases {
		if got := fieldValue(line, key); got != want {
			t.Errorf("fieldValue(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestRenderFlags(t *testing.T) {
	panes := []Pane{
		{Session: "api", ID: "%3", Path: "/ws/api", Present: true, State: "working", Since: 1700000000},
		{Session: "web", ID: "%9", Path: "/ws/web", Present: false},
		{Session: "worker", ID: "%13", Path: "/ws/worker", Present: true, State: "done", Since: 1699999000},
	}
	h := Health{
		NoPaneByCwd:     map[string]int{"/ws/worker": 5, "/ws/gone": 2}, // worker: stale; gone: no live pane -> ignored
		RecoveredByPane: map[string]int{"%3": 12},
	}
	out := Render(panes, h, 1700000060)
	for _, want := range []string{
		"Claude panes (3)",
		"not registered",            // web (not present)
		"tracking via cwd fallback", // api (%3 recovered)
		"state may be stale",        // worker (drops, no recovery)
		"0 healthy · 1 via fallback · 1 stale · 1 not registered",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Render missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "/ws/gone") {
		t.Errorf("a drop with no live pane must be ignored, not reported:\n%s", out)
	}
}

func TestRenderClean(t *testing.T) {
	panes := []Pane{{Session: "web", ID: "%32", Present: true, State: "working", Since: 1700000000}}
	out := Render(panes, Health{NoPaneByCwd: map[string]int{}, RecoveredByPane: map[string]int{}}, 1700000010)
	if !strings.Contains(out, "1 healthy · 0 via fallback · 0 stale · 0 not registered") {
		t.Errorf("clean summary wrong:\n%s", out)
	}
}

func TestParseSidebars(t *testing.T) {
	// Real list-panes output: tmux reports pane_start_command quoted.
	out := strings.Join([]string{
		"web\t%1\tagentbar\t\"/home/u/dotfiles/apps/agentbar/bin/agentbar run --theme catppuccin-mocha\"",
		"web\t%2\tclaude\t\"claude\"",
		"api\t%3\tagentbar\t\"/home/u/dotfiles/apps/agentbar/bin/agentbar run --theme solarized-light\"",
		"api\t%4\tagentbar\t\"/home/u/dotfiles/apps/agentbar/bin/agentbar run\"",
	}, "\n")
	got := ParseSidebars(out)
	want := []SidebarPane{
		{Session: "web", ID: "%1", Theme: "catppuccin-mocha"},
		{Session: "api", ID: "%3", Theme: "solarized-light"},
		{Session: "api", ID: "%4", Theme: ""},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d sidebars, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sidebar %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}

// The drift that actually happened: @agentbar-theme was set outside `theme`, so
// the file and the option disagreed and a restart silently repainted the sidebar.
func TestRenderThemeOptionSetOutsideSwitcher(t *testing.T) {
	out := RenderTheme(Theme{
		Configured: "solarized-light",
		Option:     "catppuccin-mocha",
		Sidebars:   []SidebarPane{{Session: "web", ID: "%1", Theme: "catppuccin-mocha"}},
	})
	if !strings.Contains(out, "option ≠ configured") {
		t.Errorf("drift between file and option not reported:\n%s", out)
	}
	if !strings.Contains(out, "theme solarized-light") {
		t.Errorf("remedy must name the configured flavor:\n%s", out)
	}
	if strings.Contains(out, "in sync") {
		t.Errorf("must not claim in sync:\n%s", out)
	}
}

// After `theme` fixes the option, running sidebars still render the old flavor
// until they are restarted - the state the user is left in mid-fix.
func TestRenderThemeStaleSidebars(t *testing.T) {
	out := RenderTheme(Theme{
		Configured: "solarized-light",
		Option:     "solarized-light",
		Sidebars: []SidebarPane{
			{Session: "web", ID: "%1", Theme: "catppuccin-mocha"},
			{Session: "api", ID: "%3", Theme: "catppuccin-mocha"},
			{Session: "db", ID: "%5", Theme: "solarized-light"},
		},
	})
	if !strings.Contains(out, "2 sidebar(s) render an older flavor") {
		t.Errorf("stale sidebars not counted:\n%s", out)
	}
	if !strings.Contains(out, "prefix + e twice") {
		t.Errorf("remedy missing:\n%s", out)
	}
	if strings.Contains(out, "option ≠ configured") {
		t.Errorf("file and option agree; must not flag them:\n%s", out)
	}
}

func TestRenderThemeInSync(t *testing.T) {
	out := RenderTheme(Theme{
		Configured: "solarized-light",
		Option:     "solarized-light",
		Sidebars:   []SidebarPane{{Session: "web", ID: "%1", Theme: "solarized-light"}},
	})
	if !strings.Contains(out, "✓ in sync") {
		t.Errorf("clean state should report in sync:\n%s", out)
	}
}

// No sidebar running is not drift; nothing can be stale.
func TestRenderThemeNoSidebars(t *testing.T) {
	out := RenderTheme(Theme{Configured: "solarized-light", Option: "solarized-light"})
	if !strings.Contains(out, "none running") || !strings.Contains(out, "✓ in sync") {
		t.Errorf("no sidebars should be in sync:\n%s", out)
	}
}
