package cmd

import (
	"fmt"
	"strings"
	"time"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/money"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type budgetAction string

const (
	budgetActionShow budgetAction = "show"
	budgetActionSet  budgetAction = "set"
	budgetActionQuit budgetAction = "quit"
)

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
				amountMinor, err := money.Parse(m.amountInput)
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
		s += renderActiveField("Month (YYYY-MM): ", m.monthInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, q to quit)") + "\n"
	case budgetSetStepCurrency:
		s += renderField("Month: ", m.monthInput) + "\n\n"
		s += renderActiveField("Currency: ", m.currencyInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case budgetSetStepAmount:
		s += renderField("Month: ", m.monthInput) + "\n"
		s += renderField("Currency: ", strings.ToUpper(strings.TrimSpace(m.currencyInput))) + "\n\n"
		s += renderActiveField("Budget amount: ", m.amountInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case budgetSetStepConfirm:
		s += renderField("Month: ", m.monthInput) + "\n"
		s += renderField("Currency: ", strings.ToUpper(strings.TrimSpace(m.currencyInput))) + "\n"
		s += renderField("Budget amount: ", m.amountInput) + "\n\n"
		s += warnStyle.Render("Save monthly budget?") + "\n\n"
		s += renderYesNo(m.confirmCursor == 0) + "\n"
		s += mutedStyle.Render("(use ←/→ or ↑/↓, enter to confirm, esc to go back, q to quit)") + "\n"
	case budgetSetStepDone:
		s += mutedStyle.Render("Done") + "\n"
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
	candidate, err := svc.GetBudgetCarryoverCandidate(result.monthInput, result.currencyInput, time.Now())
	if err != nil {
		return false, err
	}
	if candidate != nil {
		question := fmt.Sprintf(
			"Carry over %s %s from %s into %s?",
			money.Format(candidate.LeftMinor),
			candidate.SourceBudget.Currency,
			candidate.SourceBudget.Month,
			candidate.TargetMonth,
		)
		carryoverDecision, err = runConfirmPrompt(question)
		if err != nil {
			return false, err
		}
	}

	budget, err := svc.SetMonthlyBudgetWithCarryover(result.monthInput, result.currencyInput, result.amountInput, carryoverDecision)
	if err != nil {
		return false, err
	}

	fmt.Printf("budget set: %s (%s %s)\n", budget.Month, budget.Currency, money.Format(budget.AmountMinor))
	if candidate != nil {
		if carryoverDecision {
			fmt.Printf("carried over %s %s from %s\n", money.Format(candidate.LeftMinor), candidate.SourceBudget.Currency, candidate.SourceBudget.Month)
		} else {
			fmt.Printf("did not carry over budget from %s\n", candidate.SourceBudget.Month)
		}
	}
	return true, nil
}

func runBudgetMenuInteractive() (budgetAction, error) {
	items := []selectionItem[budgetAction]{
		{label: "Show monthly budgets", value: budgetActionShow},
		{label: "Set monthly budget", value: budgetActionSet},
		{label: "Quit", value: budgetActionQuit},
	}
	action, _, err := runSelection("Budget", "Choose an action:", items)
	if err != nil {
		return "", err
	}
	return action, nil
}

func runBudgetMonthPromptInteractive() (string, bool, error) {
	return runMonthPrompt("Show Budgets")
}

func runBudgetShow(monthInput string) error {
	month, err := coininternal.ParseBudgetMonth(monthInput, time.Now())
	if err != nil {
		return err
	}

	summaries, err := svc.GetMonthlyBudgetSummaries(monthInput, time.Now())
	if err != nil {
		return err
	}

	monthLabel := coininternal.FormatBudgetMonth(month)
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
			money.Format(summary.Budget.AmountMinor),
			money.Format(summary.Budget.RolloverMinor),
			money.Format(summary.SpentMinor),
			money.Format(summary.LeftMinor),
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

