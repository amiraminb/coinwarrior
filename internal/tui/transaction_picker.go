package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/daterange"
	"github.com/amiraminb/coinwarrior/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

type transactionLookupAction string

const (
	transactionLookupByMonth transactionLookupAction = "month"
	transactionLookupByID    transactionLookupAction = "id"
	transactionLookupQuit    transactionLookupAction = "quit"
)

type transactionListModel struct {
	title        string
	transactions []model.Transaction
	cursor       int
	selected     model.Transaction
}

func newTransactionListModel(title string, transactions []model.Transaction) transactionListModel {
	items := make([]model.Transaction, len(transactions))
	copy(items, transactions)
	return transactionListModel{title: title, transactions: items}
}

func (m transactionListModel) Init() tea.Cmd { return nil }

func (m transactionListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.transactions)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.transactions) == 0 {
				break
			}
			m.selected = m.transactions[m.cursor]
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m transactionListModel) View() string {
	s := m.title + "\n\n"
	s += "Select transaction:\n\n"

	for i, tx := range m.transactions {
		line := "  " + FormatEditableTransaction(tx)
		if i == m.cursor {
			line = focusStyle.Render("> " + FormatEditableTransaction(tx))
		}
		s += line + "\n"
	}

	s += "\n" + mutedStyle.Render("(use ↑/↓ and enter, esc to cancel, q to quit)") + "\n"
	return s
}

func SelectTransaction(title string) (model.Transaction, bool, error) {
	transactions, err := loadAllTransactionsForSelection()
	if err != nil {
		return model.Transaction{}, false, err
	}
	if len(transactions) == 0 {
		return model.Transaction{}, false, fmt.Errorf("no transactions available")
	}

	action, ok, err := runTransactionLookupMenuInteractive(title)
	if err != nil {
		return model.Transaction{}, false, err
	}
	if !ok || action == transactionLookupQuit {
		return model.Transaction{}, false, nil
	}

	switch action {
	case transactionLookupByMonth:
		monthInput, ok, err := runTransactionMonthPromptInteractive(title)
		if err != nil {
			return model.Transaction{}, false, err
		}
		if !ok {
			return model.Transaction{}, false, nil
		}

		month, err := daterange.ParseMonth(monthInput, time.Now())
		if err != nil {
			return model.Transaction{}, false, err
		}
		filtered, err := filterTransactionsByMonth(transactions, month)
		if err != nil {
			return model.Transaction{}, false, err
		}
		if len(filtered) == 0 {
			return model.Transaction{}, false, fmt.Errorf("no transactions found for %s", daterange.FormatMonth(month))
		}

		selected, ok, err := runTransactionListInteractive(title, filtered)
		if err != nil {
			return model.Transaction{}, false, err
		}
		if !ok {
			return model.Transaction{}, false, nil
		}
		return selected, true, nil
	case transactionLookupByID:
		id, ok, err := runTransactionIDPromptInteractive(title)
		if err != nil {
			return model.Transaction{}, false, err
		}
		if !ok {
			return model.Transaction{}, false, nil
		}

		for _, tx := range transactions {
			if tx.ID == strings.TrimSpace(id) {
				return tx, true, nil
			}
		}
		return model.Transaction{}, false, fmt.Errorf("transaction '%s' not found", strings.TrimSpace(id))
	default:
		return model.Transaction{}, false, nil
	}
}

func runTransactionLookupMenuInteractive(title string) (transactionLookupAction, bool, error) {
	items := []selectionItem[transactionLookupAction]{
		{label: "Show month transactions", value: transactionLookupByMonth},
		{label: "Provide transaction ID", value: transactionLookupByID},
		{label: "Cancel", value: transactionLookupQuit},
	}
	return runSelection(title, "How do you want to find the transaction?", items)
}

func runTransactionMonthPromptInteractive(title string) (string, bool, error) {
	return runMonthPrompt(title)
}

func runTransactionIDPromptInteractive(title string) (string, bool, error) {
	validate := func(s string) error {
		if s == "" {
			return fmt.Errorf("transaction id is required")
		}
		return nil
	}
	return runTextPrompt(title, "Transaction ID: ", "", validate)
}

func runTransactionListInteractive(title string, transactions []model.Transaction) (model.Transaction, bool, error) {
	p := tea.NewProgram(newTransactionListModel(title, transactions))
	finalModel, err := p.Run()
	if err != nil {
		return model.Transaction{}, false, err
	}

	result := finalModel.(transactionListModel)
	if result.selected.ID == "" {
		return model.Transaction{}, false, nil
	}
	return result.selected, true, nil
}

func loadAllTransactionsForSelection() ([]model.Transaction, error) {
	transactions, err := Repo.LoadTransactions()
	if err != nil {
		return nil, err
	}

	items := make([]model.Transaction, len(transactions))
	copy(items, transactions)
	SortTransactionsByDateDesc(items)
	return items, nil
}

func filterTransactionsByMonth(transactions []model.Transaction, month time.Time) ([]model.Transaction, error) {
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	end := start.AddDate(0, 1, -1)

	filtered := make([]model.Transaction, 0)
	for _, tx := range transactions {
		inRange, err := daterange.Contains(tx.Date, start, end)
		if err != nil {
			return nil, err
		}
		if inRange {
			filtered = append(filtered, tx)
		}
	}

	return filtered, nil
}
