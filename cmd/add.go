package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type addStep int

const (
	stepType addStep = iota
	stepAmount
	stepCurrency
	stepDone
)

type addModel struct {
	step addStep

	cursor   int
	choices  []string
	selected string

	amountInput   string
	currencyInput string
}

func newAddModel() addModel {
	return addModel{
		step:          stepType,
		cursor:        0,
		choices:       []string{"expense", "income"},
		currencyInput: "CAD",
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
		}
		switch m.step {
		case stepType:
			switch msg.String() {
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
				m.step = stepAmount
			}
		case stepAmount:
			switch msg.String() {
			case "enter":
				if m.amountInput != "" {
					m.step = stepCurrency
				}
			case "backspace":
				if len(m.amountInput) > 0 {
					m.amountInput = m.amountInput[:len(m.amountInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					ch := msg.String()
					if (ch >= "0" && ch <= "9") || ch == "." {
						m.amountInput += ch
					}
				}
			}
		case stepCurrency:
			switch msg.String() {
			case "enter":
				if m.currencyInput != "" {
					m.step = stepDone
					return m, tea.Quit
				}
			case "backspace":
				if len(m.currencyInput) > 0 {
					m.currencyInput = m.currencyInput[:len(m.currencyInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					ch := strings.ToUpper(msg.String())
					if (ch >= "a" && ch <= "z") || (ch >= "A" && ch <= "Z") {
						m.currencyInput += ch
					}
				}
			}
		}
	}
	return m, nil
}

func (m addModel) View() string {
	s := "coinw add\n\n"
	switch m.step {
	case stepType:
		s += "Select type:\n\n"
		for i, c := range m.choices {
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}
			s += prefix + c + "\n"
		}
		s += "\n(use ↑/↓ and enter, q to quit)\n"
	case stepAmount:
		s += "Type selected: " + m.selected + "\n\n"
		s += "Enter amount: " + m.amountInput + "\n"
		s += "(press enter to continue, q to quit)\n"
	case stepCurrency:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n\n"
		s += "Enter currency: " + m.currencyInput + "\n"
		s += "(press enter to continue, q to quit)\n"
	case stepDone:
		s += "Done\n"
	}
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
		if result.selected == "" || result.amountInput == "" || result.currencyInput == "" {
			fmt.Println("add cancelled")
			return nil
		}
		fmt.Printf("type: %s, amount: %s, currency: %s\n", result.selected, result.amountInput, result.currencyInput)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
