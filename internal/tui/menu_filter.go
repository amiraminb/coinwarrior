package tui

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// menuFilter keeps cursors in the original list so filtering never changes the
// meaning of a selected value or a special row such as [New category].
type menuFilter struct {
	query string
}

func (f menuFilter) indices(labels []string) []int {
	query := strings.ToLower(f.query)
	indices := make([]int, 0, len(labels))
	for i, label := range labels {
		if strings.Contains(strings.ToLower(label), query) {
			indices = append(indices, i)
		}
	}
	return indices
}

// update consumes filtering/navigation keys and blocks Enter when there are no
// matches. An empty-filter Esc is left to the enclosing form's back/cancel logic.
func (f *menuFilter) update(msg tea.KeyMsg, labels []string, cursor *int) bool {
	switch msg.Type {
	case tea.KeyUp, tea.KeyDown, tea.KeyEnter:
		indices := f.indices(labels)
		if len(indices) == 0 {
			return true
		}
		position := slices.Index(indices, *cursor)
		if position < 0 {
			position = 0
		} else if msg.Type == tea.KeyUp && position > 0 {
			position--
		} else if msg.Type == tea.KeyDown && position < len(indices)-1 {
			position++
		}
		*cursor = indices[position]
		return msg.Type != tea.KeyEnter
	case tea.KeyEsc:
		if f.query == "" {
			return false
		}
		f.query = ""
	case tea.KeyCtrlU:
		f.query = ""
	case tea.KeyBackspace, tea.KeyCtrlH:
		if f.query != "" {
			_, size := utf8.DecodeLastRuneInString(f.query)
			f.query = f.query[:len(f.query)-size]
		}
	case tea.KeyRunes, tea.KeySpace:
		if msg.Alt {
			return false
		}
		if msg.Type == tea.KeySpace {
			f.query += " "
		} else {
			for _, r := range msg.Runes {
				if unicode.IsPrint(r) {
					f.query += string(r)
				}
			}
		}
	default:
		return false
	}
	if indices := f.indices(labels); len(indices) > 0 {
		*cursor = indices[0]
	}
	return true
}

func (f menuFilter) header(labels []string) string {
	s := renderActiveField("Filter: ", f.query) + "\n\n"
	if len(f.indices(labels)) == 0 {
		s += mutedStyle.Render("No matches.") + "\n"
	}
	return s
}

func (f menuFilter) view(labels []string, cursor int) string {
	s := f.header(labels)
	for _, i := range f.indices(labels) {
		line := "  " + labels[i]
		if i == cursor {
			line = focusStyle.Render("> " + labels[i])
		}
		s += line + "\n"
	}
	return s
}

func menuHelp(escapeAction string) string {
	return "\n" + mutedStyle.Render("(type to filter, ↑/↓ and enter, backspace to edit, esc to clear/"+escapeAction+", ctrl+c to quit)") + "\n"
}

func withNewOption(items []string, label string) []string {
	return append(append([]string(nil), items...), label)
}
