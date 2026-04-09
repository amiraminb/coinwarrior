package cmd

import (
	"fmt"
	"strings"
	"time"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type budgetAction string

const (
	budgetActionShow budgetAction = "show"
	budgetActionSet  budgetAction = "set"
	budgetActionQuit budgetAction = "quit"
)

type budgetMenuChoice struct {
	label  string
	action budgetAction
}

type budgetMenuModel struct {
	choices  []budgetMenuChoice
	cursor   int
	selected budgetAction
}

type budgetMonthPromptModel struct {
	input      string
	confirmed  bool
	errMessage string
}

type budgetSetStep int

const (
	budgetSetStepMonth budgetSetStep = iota
	budgetSetStepCurrency
	budgetSetStepAmount
	budgetSetStepConfirm
	budgetSetStepDone
)

type budgetSetModel struct {
	step budgetSetStep

	monthInput    string
	currencyInput string
	amountInput   string

	confirmCursor int
	confirmed     bool
	errMessage    string
}

var (
	budgetFocusStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	budgetMutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	budgetWarnStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	budgetValueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
	budgetCursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("42")).Foreground(lipgloss.Color("0"))
)

func newBudgetSetModel() budgetSetModel {
	return budgetSetModel{
		step:          budgetSetStepMonth,
		monthInput:    coininternal.FormatBudgetMonth(time.Now()),
		currencyInput: "CAD",
	}
}

func (m budgetSetModel) Init() tea.Cmd {
	return nil
}

func (m budgetSetModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		switch m.step {
		case budgetSetStepMonth:
			switch msg.String() {
			case "enter":
				if _, err := coininternal.ParseBudgetMonth(m.monthInput, time.Now()); err != nil {
					m.errMessage = err.Error()
					break
				}
				m.errMessage = ""
				m.step = budgetSetStepCurrency
			case "backspace":
				m.errMessage = ""
				if len(m.monthInput) > 0 {
					m.monthInput = m.monthInput[:len(m.monthInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					ch := msg.String()
					if (ch >= "0" && ch <= "9") || ch == "-" {
						m.monthInput += ch
						m.errMessage = ""
					}
				}
			}
		case budgetSetStepCurrency:
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.currencyInput) == "" {
					m.errMessage = "currency is required"
					break
				}
				m.currencyInput = strings.ToUpper(strings.TrimSpace(m.currencyInput))
				m.errMessage = ""
				m.step = budgetSetStepAmount
			case "esc":
				m.errMessage = ""
				m.step = budgetSetStepMonth
			case "backspace":
				m.errMessage = ""
				if len(m.currencyInput) > 0 {
					m.currencyInput = m.currencyInput[:len(m.currencyInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.currencyInput += strings.ToUpper(msg.String())
					m.errMessage = ""
				}
			}
		case budgetSetStepAmount:
			switch msg.String() {
			case "enter":
				amountMinor, err := coininternal.ParseAmount(m.amountInput)
				if err != nil {
					m.errMessage = err.Error()
					break
				}
				if amountMinor <= 0 {
					m.errMessage = "budget amount must be greater than zero"
					break
				}
				m.errMessage = ""
				m.confirmCursor = 0
				m.step = budgetSetStepConfirm
			case "esc":
				m.errMessage = ""
				m.step = budgetSetStepCurrency
			case "backspace":
				m.errMessage = ""
				if len(m.amountInput) > 0 {
					m.amountInput = m.amountInput[:len(m.amountInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					ch := msg.String()
					if (ch >= "0" && ch <= "9") || ch == "." {
						m.amountInput += ch
						m.errMessage = ""
					}
				}
			}
		case budgetSetStepConfirm:
			switch msg.String() {
			case "left", "h", "up", "k":
				m.confirmCursor = 0
			case "right", "l", "down", "j":
				m.confirmCursor = 1
			case "enter":
				if m.confirmCursor == 0 {
					m.confirmed = true
					m.step = budgetSetStepDone
					return m, tea.Quit
				}
				m.step = budgetSetStepAmount
			case "esc":
				m.step = budgetSetStepAmount
			}
		}
	}

	return m, nil
}

func (m budgetSetModel) View() string {
	s := "Set Monthly Budget\n\n"

	switch m.step {
	case budgetSetStepMonth:
		s += renderBudgetActiveField("Month (YYYY-MM): ", m.monthInput) + "\n"
		s += renderBudgetError(m.errMessage)
		s += budgetMutedStyle.Render("(enter to continue, q to quit)") + "\n"
	case budgetSetStepCurrency:
		s += renderBudgetField("Month: ", m.monthInput) + "\n\n"
		s += renderBudgetActiveField("Currency: ", m.currencyInput) + "\n"
		s += renderBudgetError(m.errMessage)
		s += budgetMutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case budgetSetStepAmount:
		s += renderBudgetField("Month: ", m.monthInput) + "\n"
		s += renderBudgetField("Currency: ", strings.ToUpper(strings.TrimSpace(m.currencyInput))) + "\n\n"
		s += renderBudgetActiveField("Budget amount: ", m.amountInput) + "\n"
		s += renderBudgetError(m.errMessage)
		s += budgetMutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case budgetSetStepConfirm:
		s += renderBudgetField("Month: ", m.monthInput) + "\n"
		s += renderBudgetField("Currency: ", strings.ToUpper(strings.TrimSpace(m.currencyInput))) + "\n"
		s += renderBudgetField("Budget amount: ", m.amountInput) + "\n\n"
		s += budgetWarnStyle.Render("Save monthly budget?") + "\n\n"

		yes := "  Yes"
		no := "  No"
		if m.confirmCursor == 0 {
			yes = budgetFocusStyle.Render("> Yes")
		} else {
			no = budgetFocusStyle.Render("> No")
		}

		s += yes + "\n"
		s += no + "\n\n"
		s += budgetMutedStyle.Render("(use ←/→ or ↑/↓, enter to confirm, esc to go back, q to quit)") + "\n"
	case budgetSetStepDone:
		s += budgetMutedStyle.Render("Done") + "\n"
	}

	return s
}

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Manage monthly budgets",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		action, err := runBudgetMenuInteractive()
		if err != nil {
			return err
		}

		switch action {
		case budgetActionShow:
			monthInput, confirmed, err := runBudgetMonthPromptInteractive()
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Println("budget cancelled")
				return nil
			}
			return runBudgetShow(monthInput)
		case budgetActionSet:
			_, err := runBudgetSetInteractive()
			return err
		case budgetActionQuit:
			return nil
		default:
			fmt.Println("budget cancelled")
			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(budgetCmd)
}

func newBudgetMenuModel() budgetMenuModel {
	return budgetMenuModel{
		choices: []budgetMenuChoice{
			{label: "Show monthly budgets", action: budgetActionShow},
			{label: "Set monthly budget", action: budgetActionSet},
			{label: "Quit", action: budgetActionQuit},
		},
	}
}

func (m budgetMenuModel) Init() tea.Cmd {
	return nil
}

func (m budgetMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.choices[m.cursor].action
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m budgetMenuModel) View() string {
	s := "Budget\n\n"
	s += "Choose an action:\n\n"

	for i, choice := range m.choices {
		line := "  " + choice.label
		if i == m.cursor {
			line = budgetFocusStyle.Render("> " + choice.label)
		}
		s += line + "\n"
	}

	s += "\n" + budgetMutedStyle.Render("(use ↑/↓ and enter, esc to cancel, q to quit)") + "\n"
	return s
}

func newBudgetMonthPromptModel() budgetMonthPromptModel {
	return budgetMonthPromptModel{input: coininternal.FormatBudgetMonth(time.Now())}
}

func (m budgetMonthPromptModel) Init() tea.Cmd {
	return nil
}

func (m budgetMonthPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m budgetMonthPromptModel) View() string {
	s := "Show Budgets\n\n"
	s += renderBudgetActiveField("Month (YYYY-MM): ", m.input) + "\n"
	s += renderBudgetError(m.errMessage)
	s += budgetMutedStyle.Render("(enter to continue, esc to cancel, q to quit)") + "\n"
	return s
}

func runBudgetSetInteractive() (bool, error) {
	p := tea.NewProgram(newBudgetSetModel())

	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}

	result := finalModel.(budgetSetModel)
	if !result.confirmed {
		fmt.Println("budget set cancelled")
		return false, nil
	}

	carryoverDecision := false
	candidate, err := coininternal.GetBudgetCarryoverCandidate(result.monthInput, result.currencyInput, time.Now())
	if err != nil {
		return false, err
	}
	if candidate != nil {
		question := fmt.Sprintf(
			"Carry over %s %s from %s into %s?",
			coininternal.FormatMinor(candidate.LeftMinor),
			candidate.SourceBudget.Currency,
			candidate.SourceBudget.Month,
			candidate.TargetMonth,
		)
		carryoverDecision, err = runConfirmPrompt(question)
		if err != nil {
			return false, err
		}
	}

	budget, err := coininternal.SetMonthlyBudgetWithCarryover(result.monthInput, result.currencyInput, result.amountInput, carryoverDecision)
	if err != nil {
		return false, err
	}

	fmt.Printf("budget set: %s (%s %s)\n", budget.Month, budget.Currency, coininternal.FormatMinor(budget.AmountMinor))
	if candidate != nil {
		if carryoverDecision {
			fmt.Printf("carried over %s %s from %s\n", coininternal.FormatMinor(candidate.LeftMinor), candidate.SourceBudget.Currency, candidate.SourceBudget.Month)
		} else {
			fmt.Printf("did not carry over budget from %s\n", candidate.SourceBudget.Month)
		}
	}
	return true, nil
}

func runBudgetMenuInteractive() (budgetAction, error) {
	p := tea.NewProgram(newBudgetMenuModel())

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(budgetMenuModel)
	return result.selected, nil
}

func runBudgetMonthPromptInteractive() (string, bool, error) {
	p := tea.NewProgram(newBudgetMonthPromptModel())

	finalModel, err := p.Run()
	if err != nil {
		return "", false, err
	}

	result := finalModel.(budgetMonthPromptModel)
	if !result.confirmed {
		return "", false, nil
	}

	return strings.TrimSpace(result.input), true, nil
}

func runBudgetShow(monthInput string) error {
	month, err := coininternal.ParseBudgetMonth(monthInput, time.Now())
	if err != nil {
		return err
	}

	summaries, err := coininternal.GetMonthlyBudgetSummaries(monthInput, time.Now())
	if err != nil {
		return err
	}

	monthLabel := coininternal.FormatBudgetMonth(month)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("250"))
	fmt.Println(headerStyle.Render("budget " + monthLabel))
	fmt.Println()

	if len(summaries) == 0 {
		fmt.Printf("no budgets for %s\n", monthLabel)
		return nil
	}

	rows := make([]table.Row, 0, len(summaries))
	for _, summary := range summaries {
		rows = append(rows, table.Row{
			summary.Budget.Currency,
			coininternal.FormatMinor(summary.Budget.AmountMinor),
			coininternal.FormatMinor(summary.Budget.RolloverMinor),
			coininternal.FormatMinor(summary.SpentMinor),
			coininternal.FormatMinor(summary.LeftMinor),
			summary.Status,
		})
	}

	renderTable(
		[]table.Column{
			{Title: "CUR", Width: 5},
			{Title: "BUDGET", Width: 14},
			{Title: "ROLL", Width: 14},
			{Title: "SPENT", Width: 14},
			{Title: "LEFT", Width: 14},
			{Title: "STATUS", Width: 10},
		},
		rows,
	)

	return nil
}

func renderBudgetField(label, value string) string {
	return label + budgetValueStyle.Render(value)
}

func renderBudgetActiveField(label, value string) string {
	return label + renderBudgetCursor(value)
}

func renderBudgetCursor(value string) string {
	rendered := ""
	if value != "" {
		rendered = budgetValueStyle.Render(value)
	}

	return rendered + budgetCursorStyle.Render(" ")
}

func renderBudgetError(message string) string {
	if message == "" {
		return ""
	}

	return budgetWarnStyle.Render(message) + "\n"
}
