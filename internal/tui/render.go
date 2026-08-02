package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

func RenderTable(columns []table.Column, rows []table.Row) {
	RenderStyledTable(columns, rows, nil)
}

// The table renders every cell Inline, which strips styling from the cell text,
// so a per-row style has to be applied to the finished line instead.
func RenderStyledTable(columns []table.Column, rows []table.Row, rowStyle func(int) lipgloss.Style) {
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

	view := t.View()
	if rowStyle == nil {
		fmt.Println(view)
		return
	}

	lines := strings.Split(view, "\n")
	dataStart := len(lines) - len(rows)
	for i := dataStart; i < len(lines); i++ {
		lines[i] = rowStyle(i - dataStart).Render(stripANSI(lines[i]))
	}
	fmt.Println(strings.Join(lines, "\n"))
}

var ansiPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
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

func renderYesNo(yesFocused bool) string {
	yes := "  Yes"
	no := "  No"
	if yesFocused {
		yes = focusStyle.Render("> Yes")
	} else {
		no = focusStyle.Render("> No")
	}
	return yes + "\n" + no + "\n"
}
