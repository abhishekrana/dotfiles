package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/abhishekrana/tmux-agent-sidebar/internal/model"
)

// The five-state language: permission is a hard red block, distinct from the amber ask.
func TestStateColorFiveStates(t *testing.T) {
	th := SolarizedLight()
	want := map[model.AgentState]lipgloss.Color{
		model.StateWorking:    th.Working,
		model.StatePermission: th.Blocked,
		model.StateQuestion:   th.Asking,
		model.StateDone:       th.Done,
		model.StateIdle:       th.Muted,
	}
	for st, w := range want {
		if got := th.StateColor(st); got != w {
			t.Errorf("StateColor(%q) = %v, want %v", st, got, w)
		}
	}
	// Regression on the old amber==amber: blocked and asking must be distinct.
	if th.Blocked == th.Asking {
		t.Error("blocked and asking must be distinct colors")
	}
}

// All four flavors resolve; unknown/empty falls back to the default (Solarized Light).
func TestThemeByNameFlavors(t *testing.T) {
	if ThemeByName("catppuccin-mocha").Working != CatppuccinMocha().Working {
		t.Error("catppuccin-mocha should resolve to the Mocha palette")
	}
	if ThemeByName("solarized-dark").SelBg != SolarizedDark().SelBg {
		t.Error("solarized-dark should resolve to the Solarized Dark palette")
	}
	if ThemeByName("catppuccin-latte").Accent != CatppuccinLatte().Accent {
		t.Error("catppuccin-latte should resolve to the Latte palette")
	}
	if ThemeByName("nonesuch").Accent != SolarizedLight().Accent {
		t.Error("unknown theme should fall back to Solarized Light")
	}
}
