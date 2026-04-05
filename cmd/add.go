package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type addModel struct {
	cursor   int
	choices  []string
	selected string
	done     bool
}

func newAddModel() addModel {
	return addModel{
		cursor:  0,
		choices: []string{"expense", "income"},
	}
}

func (m addModel) Init() tea.Cmd {
	return nil
}

func (m addModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.choices[m.cursor]
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m addModel) View() string {
	if m.done {
		return "selected type: " + m.selected + "\n"
	}
	s := "coinw add\n\nSelect type:\n\n"
	for i, c := range m.choices {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		s += prefix + c + "\n"
	}
	s += "\n(use ↑/↓ and enter, q to quit)\n"
	return s
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a transaction",
	RunE: func(cmd *cobra.Command, args []string) error {
		m := newAddModel()
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		result := finalModel.(addModel)
		if result.selected == "" {
			fmt.Println("add cancelled")
			return nil
		}
		fmt.Printf("type selected: %s\n", result.selected)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
