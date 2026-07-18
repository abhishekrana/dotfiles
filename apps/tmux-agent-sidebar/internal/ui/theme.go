package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/abhishekrana/tmux-agent-sidebar/internal/model"
)

// Theme is the sidebar palette. Values mirror ~/dotfiles/design/palette.toml (the single
// source of truth). No explicit bg fill, so the sidebar blends into the terminal background.
type Theme struct {
	Fg       lipgloss.Color // default text
	Muted    lipgloss.Color // separators, hints, idle
	Emphasis lipgloss.Color // session names, headlines
	Accent   lipgloss.Color // selection rail, current marker
	SelBg    lipgloss.Color // selected row background
	Working  lipgloss.Color // calm, cool — the common case
	Asking   lipgloss.Color // amber — a soft question needs you
	Blocked  lipgloss.Color // red — a hard stop needs you
	Done     lipgloss.Color // green — ready to review
}

// SolarizedLight is the default flavor.
func SolarizedLight() Theme {
	return Theme{
		Fg:       "#657b83",
		Muted:    "#93a1a1",
		Emphasis: "#586e75",
		Accent:   "#268bd2",
		SelBg:    "#eee8d5",
		Working:  "#2aa198",
		Asking:   "#b58900",
		Blocked:  "#dc322f",
		Done:     "#859900",
	}
}

// SolarizedDark shares Solarized's accents; only the base tones invert.
func SolarizedDark() Theme {
	return Theme{
		Fg:       "#839496",
		Muted:    "#586e75",
		Emphasis: "#93a1a1",
		Accent:   "#268bd2",
		SelBg:    "#073642",
		Working:  "#2aa198",
		Asking:   "#b58900",
		Blocked:  "#dc322f",
		Done:     "#859900",
	}
}

// CatppuccinLatte is the light Catppuccin flavor.
func CatppuccinLatte() Theme {
	return Theme{
		Fg:       "#4c4f69",
		Muted:    "#8c8fa1",
		Emphasis: "#2e3047",
		Accent:   "#1e66f5",
		SelBg:    "#ccd0da",
		Working:  "#179299",
		Asking:   "#df8e1d",
		Blocked:  "#d20f39",
		Done:     "#40a02b",
	}
}

// CatppuccinMocha is the dark Catppuccin flavor.
func CatppuccinMocha() Theme {
	return Theme{
		Fg:       "#cdd6f4",
		Muted:    "#6c7086",
		Emphasis: "#eef1fb",
		Accent:   "#89b4fa",
		SelBg:    "#313244",
		Working:  "#94e2d5",
		Asking:   "#fab387",
		Blocked:  "#f38ba8",
		Done:     "#a6e3a1",
	}
}

// ThemeByName resolves the @agent-sidebar-theme option value; unknown falls back to the default.
func ThemeByName(name string) Theme {
	switch name {
	case "solarized-dark":
		return SolarizedDark()
	case "catppuccin-latte":
		return CatppuccinLatte()
	case "catppuccin-mocha":
		return CatppuccinMocha()
	case "dark": // back-compat: the old generic dark maps to Solarized Dark
		return SolarizedDark()
	default:
		return SolarizedLight()
	}
}

// StateColor maps an agent state to its theme color. Permission is the hard red block;
// question is the soft amber ask — the five-state language (idle reuses Muted).
func (t Theme) StateColor(s model.AgentState) lipgloss.Color {
	switch s {
	case model.StateWorking:
		return t.Working
	case model.StatePermission:
		return t.Blocked
	case model.StateQuestion:
		return t.Asking
	case model.StateDone:
		return t.Done
	default:
		return t.Muted
	}
}
