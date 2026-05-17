package cmd

import "github.com/charmbracelet/lipgloss"

var (
	focusStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	mutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	warnStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	valueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
	cursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("42")).Foreground(lipgloss.Color("0"))
)
