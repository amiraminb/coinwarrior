package cmd

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

func renderTable(columns []table.Column, rows []table.Row) {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(rows)+1),
	)

	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	styles.Cell = styles.Cell.Foreground(lipgloss.Color("252"))
	t.SetStyles(styles)

	fmt.Println(t.View())
}

func renderField(label, value string) string {
	return label + valueStyle.Render(value)
}

func renderActiveField(label, value string) string {
	rendered := ""
	if value != "" {
		rendered = valueStyle.Render(value)
	}
	return label + rendered + cursorStyle.Render(" ")
}

func renderError(message string) string {
	if message == "" {
		return ""
	}
	return warnStyle.Render(message) + "\n"
}
