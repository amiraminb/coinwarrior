package cmd

import (
	"strings"
	"time"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	tea "github.com/charmbracelet/bubbletea"
)

type confirmModel struct {
	question string
	cursor   int
	answer   bool
}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "left", "h", "up", "k":
			m.cursor = 0
		case "right", "l", "down", "j":
			m.cursor = 1
		case "enter":
			m.answer = m.cursor == 0
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m confirmModel) View() string {
	s := m.question + "\n\n"

	yes := "  Yes"
	no := "  No"
	if m.cursor == 0 {
		yes = focusStyle.Render("> Yes")
	} else {
		no = focusStyle.Render("> No")
	}

	s += yes + "\n"
	s += no + "\n\n"
	s += mutedStyle.Render("(use ←/→ or ↑/↓ and enter)") + "\n"

	return s
}

func runConfirmPrompt(question string) (bool, error) {
	p := tea.NewProgram(confirmModel{question: question})

	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}

	result := finalModel.(confirmModel)
	return result.answer, nil
}

type selectionItem[T comparable] struct {
	label string
	value T
}

type selectionModel[T comparable] struct {
	title    string
	prompt   string
	items    []selectionItem[T]
	cursor   int
	chose    bool
	selected T
}

func newSelectionModel[T comparable](title, prompt string, items []selectionItem[T]) selectionModel[T] {
	return selectionModel[T]{title: title, prompt: prompt, items: items}
}

func (m selectionModel[T]) Init() tea.Cmd { return nil }

func (m selectionModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.items) == 0 {
				return m, tea.Quit
			}
			m.selected = m.items[m.cursor].value
			m.chose = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m selectionModel[T]) View() string {
	s := ""
	if m.title != "" {
		s += m.title + "\n\n"
	}
	if m.prompt != "" {
		s += m.prompt + "\n\n"
	}

	for i, item := range m.items {
		line := "  " + item.label
		if i == m.cursor {
			line = focusStyle.Render("> " + item.label)
		}
		s += line + "\n"
	}

	s += "\n" + mutedStyle.Render("(use ↑/↓ and enter, esc to cancel, q to quit)") + "\n"
	return s
}

func runSelection[T comparable](title, prompt string, items []selectionItem[T]) (T, bool, error) {
	p := tea.NewProgram(newSelectionModel(title, prompt, items))

	finalModel, err := p.Run()
	if err != nil {
		var zero T
		return zero, false, err
	}

	result := finalModel.(selectionModel[T])
	if !result.chose {
		var zero T
		return zero, false, nil
	}
	return result.selected, true, nil
}

type monthPromptModel struct {
	title      string
	input      string
	confirmed  bool
	errMessage string
}

func newMonthPromptModel(title string) monthPromptModel {
	return monthPromptModel{
		title: title,
		input: coininternal.FormatBudgetMonth(time.Now()),
	}
}

func (m monthPromptModel) Init() tea.Cmd { return nil }

func (m monthPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if _, err := coininternal.ParseBudgetMonth(m.input, time.Now()); err != nil {
				m.errMessage = err.Error()
				return m, nil
			}
			m.confirmed = true
			return m, tea.Quit
		case "esc":
			return m, tea.Quit
		case "backspace":
			m.errMessage = ""
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(msg.String()) == 1 {
				ch := msg.String()
				if (ch >= "0" && ch <= "9") || ch == "-" {
					m.input += ch
					m.errMessage = ""
				}
			}
		}
	}

	return m, nil
}

func (m monthPromptModel) View() string {
	s := ""
	if m.title != "" {
		s += m.title + "\n\n"
	}
	s += "Month (YYYY-MM): " + valueStyle.Render(m.input) + cursorStyle.Render(" ") + "\n"
	if m.errMessage != "" {
		s += warnStyle.Render(m.errMessage) + "\n"
	}
	s += mutedStyle.Render("(enter to continue, esc to cancel, q to quit)") + "\n"
	return s
}

func runMonthPrompt(title string) (string, bool, error) {
	p := tea.NewProgram(newMonthPromptModel(title))

	finalModel, err := p.Run()
	if err != nil {
		return "", false, err
	}

	result := finalModel.(monthPromptModel)
	if !result.confirmed {
		return "", false, nil
	}
	return strings.TrimSpace(result.input), true, nil
}
