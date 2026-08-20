package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/abhishekrana/agentbar/internal/model"
)

// Theme is the sidebar palette. The flavors live in theme_gen.go, generated from
// design/palette.toml - the single source. No explicit bg fill, so the sidebar
// blends into the terminal background.
type Theme struct {
	Fg       lipgloss.Color // default text
	Muted    lipgloss.Color // separators, hints, idle
	Emphasis lipgloss.Color // session names, headlines
	Accent   lipgloss.Color // selection rail, current marker
	SelBg    lipgloss.Color // selected row background
	Working  lipgloss.Color // calm, cool - the common case
	Asking   lipgloss.Color // amber - a soft question needs you
	Blocked  lipgloss.Color // red - a hard stop needs you
	Done     lipgloss.Color // green - ready to review
}

// StateColor maps an agent state to its theme color. Permission is the hard red block;
// question is the soft amber ask - the five-state language (idle reuses Muted).
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
