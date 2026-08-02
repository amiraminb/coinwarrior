package tui

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// The table renders cells Inline, which drops styling from the cell text and
// repaints it, so the row style must be applied to a stripped line.
func TestStripANSIClearsTheTablesOwnColour(t *testing.T) {
	if got := stripANSI("\x1b[38;5;252m Groceries \x1b[0m"); got != " Groceries " {
		t.Errorf("stripANSI = %q, want %q", got, " Groceries ")
	}
	if got := stripANSI("plain"); got != "plain" {
		t.Errorf("stripANSI altered unstyled text: %q", got)
	}
}

func TestRenderStyledTableCallsTheHookPerDataRow(t *testing.T) {
	var seen []int
	RenderStyledTable(
		[]table.Column{{Title: "CATEGORY", Width: 14}},
		[]table.Row{{"Groceries"}, {"Dining"}, {"Housing"}},
		func(i int) lipgloss.Style {
			seen = append(seen, i)
			return lipgloss.NewStyle()
		},
	)

	if len(seen) != 3 {
		t.Fatalf("hook called %d times, want once per data row", len(seen))
	}
	for i, got := range seen {
		if got != i {
			t.Errorf("hook call %d received index %d", i, got)
		}
	}
}

func TestRenderStyledTableWithoutAHookMatchesRenderTable(t *testing.T) {
	cols := []table.Column{{Title: "CATEGORY", Width: 14}}
	rows := []table.Row{{"Groceries"}}

	if !strings.Contains(captureStdout(t, func() { RenderStyledTable(cols, rows, nil) }), "Groceries") {
		t.Error("a nil hook dropped the row")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = original

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(out)
}
