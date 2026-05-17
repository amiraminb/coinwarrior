package cmd

import (
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/daterange"
	"github.com/charmbracelet/bubbles/textinput"
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
	s += renderYesNo(m.cursor == 0) + "\n"
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

type textPromptModel struct {
	title      string
	prompt     string
	input      textinput.Model
	validate   func(string) error
	confirmed  bool
	errMessage string
}

func newTextPromptModel(title, prompt, initial string, validate func(string) error) textPromptModel {
	ti := textinput.New()
	ti.Prompt = prompt
	ti.SetValue(initial)
	ti.Focus()
	return textPromptModel{title: title, prompt: prompt, input: ti, validate: validate}
}

func (m textPromptModel) Init() tea.Cmd { return textinput.Blink }

func (m textPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, tea.Quit
		case "enter":
			if m.validate != nil {
				if err := m.validate(strings.TrimSpace(m.input.Value())); err != nil {
					m.errMessage = err.Error()
					return m, nil
				}
			}
			m.confirmed = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.errMessage = ""
	return m, cmd
}

func (m textPromptModel) View() string {
	s := ""
	if m.title != "" {
		s += m.title + "\n\n"
	}
	s += m.input.View() + "\n"
	if m.errMessage != "" {
		s += warnStyle.Render(m.errMessage) + "\n"
	}
	s += mutedStyle.Render("(enter to continue, esc to cancel, ctrl+c to quit)") + "\n"
	return s
}

func runTextPrompt(title, prompt, initial string, validate func(string) error) (string, bool, error) {
	p := tea.NewProgram(newTextPromptModel(title, prompt, initial, validate))

	finalModel, err := p.Run()
	if err != nil {
		return "", false, err
	}

	result := finalModel.(textPromptModel)
	if !result.confirmed {
		return "", false, nil
	}
	return strings.TrimSpace(result.input.Value()), true, nil
}

func runMonthPrompt(title string) (string, bool, error) {
	validate := func(s string) error {
		_, err := daterange.ParseMonth(s, time.Now())
		return err
	}
	return runTextPrompt(title, "Month (YYYY-MM): ", daterange.FormatMonth(time.Now()), validate)
}
