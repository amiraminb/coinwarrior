package cmd

import (
	"fmt"
	"strings"
	"time"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/domain"
	"github.com/amiraminb/coinwarrior/internal/repository"
	tea "github.com/charmbracelet/bubbletea"
)

type transactionLookupAction string

const (
	transactionLookupByMonth transactionLookupAction = "month"
	transactionLookupByID    transactionLookupAction = "id"
	transactionLookupQuit    transactionLookupAction = "quit"
)

type transactionLookupChoice struct {
	label  string
	action transactionLookupAction
}

type transactionLookupMenuModel struct {
	title    string
	choices  []transactionLookupChoice
	cursor   int
	selected transactionLookupAction
}

type transactionMonthPromptModel struct {
	title      string
	input      string
	confirmed  bool
	errMessage string
}

type transactionIDPromptModel struct {
	title      string
	input      string
	confirmed  bool
	errMessage string
}

type transactionListModel struct {
	title        string
	transactions []domain.Transaction
	cursor       int
	selected     domain.Transaction
}

func newTransactionLookupMenuModel(title string) transactionLookupMenuModel {
	return transactionLookupMenuModel{
		title: title,
		choices: []transactionLookupChoice{
			{label: "Show month transactions", action: transactionLookupByMonth},
			{label: "Provide transaction ID", action: transactionLookupByID},
			{label: "Cancel", action: transactionLookupQuit},
		},
	}
}

func newTransactionMonthPromptModel(title string) transactionMonthPromptModel {
	return transactionMonthPromptModel{
		title: title,
		input: coininternal.FormatBudgetMonth(time.Now()),
	}
}

func newTransactionIDPromptModel(title string) transactionIDPromptModel {
	return transactionIDPromptModel{title: title}
}

func newTransactionListModel(title string, transactions []domain.Transaction) transactionListModel {
	items := make([]domain.Transaction, len(transactions))
	copy(items, transactions)
	return transactionListModel{title: title, transactions: items}
}

func (m transactionLookupMenuModel) Init() tea.Cmd  { return nil }
func (m transactionMonthPromptModel) Init() tea.Cmd { return nil }
func (m transactionIDPromptModel) Init() tea.Cmd    { return nil }
func (m transactionListModel) Init() tea.Cmd        { return nil }

func (m transactionLookupMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m transactionMonthPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m transactionIDPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if strings.TrimSpace(m.input) == "" {
				m.errMessage = "transaction id is required"
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
				m.input += msg.String()
				m.errMessage = ""
			}
		}
	}

	return m, nil
}

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

func (m transactionLookupMenuModel) View() string {
	s := m.title + "\n\n"
	s += "How do you want to find the transaction?\n\n"

	for i, choice := range m.choices {
		line := "  " + choice.label
		if i == m.cursor {
			line = editFocusStyle.Render("> " + choice.label)
		}
		s += line + "\n"
	}

	s += "\n" + editMutedStyle.Render("(use ↑/↓ and enter, esc to cancel, q to quit)") + "\n"
	return s
}

func (m transactionMonthPromptModel) View() string {
	s := m.title + "\n\n"
	s += renderActiveEditField("Month (YYYY-MM): ", m.input) + "\n"
	s += renderEditError(m.errMessage)
	s += editMutedStyle.Render("(enter to continue, esc to cancel, q to quit)") + "\n"
	return s
}

func (m transactionIDPromptModel) View() string {
	s := m.title + "\n\n"
	s += renderActiveEditField("Transaction ID: ", m.input) + "\n"
	s += renderEditError(m.errMessage)
	s += editMutedStyle.Render("(enter to continue, esc to cancel, q to quit)") + "\n"
	return s
}

func (m transactionListModel) View() string {
	s := m.title + "\n\n"
	s += "Select transaction:\n\n"

	for i, tx := range m.transactions {
		line := "  " + formatEditableTransaction(tx)
		if i == m.cursor {
			line = editFocusStyle.Render("> " + formatEditableTransaction(tx))
		}
		s += line + "\n"
	}

	s += "\n" + editMutedStyle.Render("(use ↑/↓ and enter, esc to cancel, q to quit)") + "\n"
	return s
}

func selectTransactionInteractive(title string) (domain.Transaction, bool, error) {
	transactions, err := loadAllTransactionsForSelection()
	if err != nil {
		return domain.Transaction{}, false, err
	}
	if len(transactions) == 0 {
		return domain.Transaction{}, false, fmt.Errorf("no transactions available")
	}

	action, ok, err := runTransactionLookupMenuInteractive(title)
	if err != nil {
		return domain.Transaction{}, false, err
	}
	if !ok || action == transactionLookupQuit {
		return domain.Transaction{}, false, nil
	}

	switch action {
	case transactionLookupByMonth:
		monthInput, ok, err := runTransactionMonthPromptInteractive(title)
		if err != nil {
			return domain.Transaction{}, false, err
		}
		if !ok {
			return domain.Transaction{}, false, nil
		}

		month, err := coininternal.ParseBudgetMonth(monthInput, time.Now())
		if err != nil {
			return domain.Transaction{}, false, err
		}
		filtered, err := filterTransactionsByMonth(transactions, month)
		if err != nil {
			return domain.Transaction{}, false, err
		}
		if len(filtered) == 0 {
			return domain.Transaction{}, false, fmt.Errorf("no transactions found for %s", coininternal.FormatBudgetMonth(month))
		}

		selected, ok, err := runTransactionListInteractive(title, filtered)
		if err != nil {
			return domain.Transaction{}, false, err
		}
		if !ok {
			return domain.Transaction{}, false, nil
		}
		return selected, true, nil
	case transactionLookupByID:
		id, ok, err := runTransactionIDPromptInteractive(title)
		if err != nil {
			return domain.Transaction{}, false, err
		}
		if !ok {
			return domain.Transaction{}, false, nil
		}

		for _, tx := range transactions {
			if tx.ID == strings.TrimSpace(id) {
				return tx, true, nil
			}
		}
		return domain.Transaction{}, false, fmt.Errorf("transaction '%s' not found", strings.TrimSpace(id))
	default:
		return domain.Transaction{}, false, nil
	}
}

func runTransactionLookupMenuInteractive(title string) (transactionLookupAction, bool, error) {
	p := tea.NewProgram(newTransactionLookupMenuModel(title))
	finalModel, err := p.Run()
	if err != nil {
		return "", false, err
	}

	result := finalModel.(transactionLookupMenuModel)
	if result.selected == "" {
		return "", false, nil
	}
	return result.selected, true, nil
}

func runTransactionMonthPromptInteractive(title string) (string, bool, error) {
	p := tea.NewProgram(newTransactionMonthPromptModel(title))
	finalModel, err := p.Run()
	if err != nil {
		return "", false, err
	}

	result := finalModel.(transactionMonthPromptModel)
	if !result.confirmed {
		return "", false, nil
	}
	return strings.TrimSpace(result.input), true, nil
}

func runTransactionIDPromptInteractive(title string) (string, bool, error) {
	p := tea.NewProgram(newTransactionIDPromptModel(title))
	finalModel, err := p.Run()
	if err != nil {
		return "", false, err
	}

	result := finalModel.(transactionIDPromptModel)
	if !result.confirmed {
		return "", false, nil
	}
	return strings.TrimSpace(result.input), true, nil
}

func runTransactionListInteractive(title string, transactions []domain.Transaction) (domain.Transaction, bool, error) {
	p := tea.NewProgram(newTransactionListModel(title, transactions))
	finalModel, err := p.Run()
	if err != nil {
		return domain.Transaction{}, false, err
	}

	result := finalModel.(transactionListModel)
	if result.selected.ID == "" {
		return domain.Transaction{}, false, nil
	}
	return result.selected, true, nil
}

func loadAllTransactionsForSelection() ([]domain.Transaction, error) {
	transactions, err := repository.FRepository.LoadTransactions()
	if err != nil {
		return nil, err
	}

	items := make([]domain.Transaction, len(transactions))
	copy(items, transactions)
	sortEditableTransactions(items)
	return items, nil
}

func filterTransactionsByMonth(transactions []domain.Transaction, month time.Time) ([]domain.Transaction, error) {
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	end := start.AddDate(0, 1, -1)

	filtered := make([]domain.Transaction, 0)
	for _, tx := range transactions {
		inRange, err := coininternal.TransactionInRange(tx.Date, start, end)
		if err != nil {
			return nil, err
		}
		if inRange {
			filtered = append(filtered, tx)
		}
	}

	return filtered, nil
}
