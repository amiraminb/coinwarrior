package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type editStep int

const (
	editStepSelectTransaction editStep = iota
	editStepDate
	editStepAmount
	editStepCategory
	editStepAccount
	editStepToAccount
	editStepNote
	editStepConfirm
	editStepDone
)

type editModel struct {
	step editStep

	transactions []model.Transaction
	cursor       int

	selected       model.Transaction
	dateInput      string
	amountInput    string
	categoryInput  string
	accountInput   string
	toAccountInput string
	noteInput      string

	confirmCursor int
	confirmed     bool
	errMessage    string
}

var (
	editFocusStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	editMutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	editWarnStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	editValueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
	editCursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("42")).Foreground(lipgloss.Color("0"))
)

func newEditModel(transactions []model.Transaction) editModel {
	items := make([]model.Transaction, len(transactions))
	copy(items, transactions)

	return editModel{
		step:         editStepSelectTransaction,
		transactions: items,
	}
}

func (m editModel) Init() tea.Cmd {
	return nil
}

func (m editModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		switch m.step {
		case editStepSelectTransaction:
			switch msg.String() {
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
				m.dateInput = m.selected.Date
				m.amountInput = formatEditAmountInput(m.selected.AmountMinor)
				m.categoryInput = m.selected.Category
				m.accountInput = m.selected.Account
				m.toAccountInput = m.selected.ToAccount
				m.noteInput = m.selected.Note
				m.errMessage = ""
				m.step = editStepDate
			}
		case editStepDate:
			switch msg.String() {
			case "enter":
				date := strings.TrimSpace(m.dateInput)
				if date == "" {
					m.errMessage = "date is required"
					break
				}
				if _, err := time.Parse("2006-01-02", date); err != nil {
					m.errMessage = fmt.Sprintf("invalid date format: %s", date)
					break
				}
				m.errMessage = ""
				m.step = editStepAmount
			case "esc":
				m.errMessage = ""
				m.step = editStepSelectTransaction
			case "backspace":
				m.errMessage = ""
				if len(m.dateInput) > 0 {
					m.dateInput = m.dateInput[:len(m.dateInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					ch := msg.String()
					if (ch >= "0" && ch <= "9") || ch == "-" {
						m.dateInput += ch
						m.errMessage = ""
					}
				}
			}
		case editStepAmount:
			switch msg.String() {
			case "enter":
				amount := strings.TrimSpace(m.amountInput)
				amountMinor, err := coininternal.ParseAmount(amount)
				if err != nil {
					m.errMessage = err.Error()
					break
				}
				if amountMinor <= 0 {
					m.errMessage = "amount must be greater than zero"
					break
				}
				m.errMessage = ""
				m.step = editStepCategory
			case "esc":
				m.errMessage = ""
				m.step = editStepDate
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
		case editStepCategory:
			switch msg.String() {
			case "enter":
				m.errMessage = ""
				m.step = editStepAccount
			case "esc":
				m.errMessage = ""
				m.step = editStepAmount
			case "backspace":
				if len(m.categoryInput) > 0 {
					m.categoryInput = m.categoryInput[:len(m.categoryInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.categoryInput += msg.String()
				}
			}
		case editStepAccount:
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.accountInput) == "" {
					m.errMessage = "account is required"
					break
				}
				m.errMessage = ""
				if m.selected.Type == coininternal.TransactionTypeTransfer {
					m.step = editStepToAccount
				} else {
					m.step = editStepNote
				}
			case "esc":
				m.errMessage = ""
				m.step = editStepCategory
			case "backspace":
				m.errMessage = ""
				if len(m.accountInput) > 0 {
					m.accountInput = m.accountInput[:len(m.accountInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.accountInput += msg.String()
					m.errMessage = ""
				}
			}
		case editStepToAccount:
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.toAccountInput) == "" {
					m.errMessage = "destination account is required"
					break
				}
				if strings.EqualFold(strings.TrimSpace(m.accountInput), strings.TrimSpace(m.toAccountInput)) {
					m.errMessage = "source and destination accounts must be different"
					break
				}
				m.errMessage = ""
				m.step = editStepNote
			case "esc":
				m.errMessage = ""
				m.step = editStepAccount
			case "backspace":
				m.errMessage = ""
				if len(m.toAccountInput) > 0 {
					m.toAccountInput = m.toAccountInput[:len(m.toAccountInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.toAccountInput += msg.String()
					m.errMessage = ""
				}
			}
		case editStepNote:
			switch msg.String() {
			case "enter":
				m.errMessage = ""
				m.confirmCursor = 0
				m.step = editStepConfirm
			case "esc":
				m.errMessage = ""
				if m.selected.Type == coininternal.TransactionTypeTransfer {
					m.step = editStepToAccount
				} else {
					m.step = editStepAccount
				}
			case "backspace":
				if len(m.noteInput) > 0 {
					m.noteInput = m.noteInput[:len(m.noteInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.noteInput += msg.String()
				}
			}
		case editStepConfirm:
			switch msg.String() {
			case "left", "h", "up", "k":
				m.confirmCursor = 0
			case "right", "l", "down", "j":
				m.confirmCursor = 1
			case "enter":
				if m.confirmCursor == 0 {
					m.confirmed = true
					m.step = editStepDone
					return m, tea.Quit
				}
				m.step = editStepNote
			case "esc":
				m.step = editStepNote
			}
		}
	}

	return m, nil
}

func (m editModel) View() string {
	s := "Edit Transaction\n\n"

	switch m.step {
	case editStepSelectTransaction:
		s += "Select transaction:\n\n"
		for i, tx := range m.transactions {
			line := "  " + formatEditableTransaction(tx)
			if i == m.cursor {
				line = editFocusStyle.Render("> " + formatEditableTransaction(tx))
			}
			s += line + "\n"
		}
		s += "\n" + editMutedStyle.Render("(use ↑/↓ and enter, q to quit)") + "\n"
	case editStepDate:
		s += renderEditField("Editing: ", m.selected.ID) + "\n"
		s += renderEditField("Type: ", m.selected.Type) + "\n\n"
		s += renderActiveEditField("Date: ", m.dateInput) + "\n"
		s += renderEditError(m.errMessage)
		s += editMutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case editStepAmount:
		s += renderEditField("Editing: ", m.selected.ID) + "\n"
		s += renderEditField("Date: ", m.dateInput) + "\n\n"
		s += renderActiveEditField("Amount: ", m.amountInput) + "\n"
		s += renderEditError(m.errMessage)
		s += editMutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case editStepCategory:
		s += renderEditField("Editing: ", m.selected.ID) + "\n"
		s += renderEditField("Date: ", m.dateInput) + "\n"
		s += renderEditField("Amount: ", m.amountInput) + "\n\n"
		s += renderActiveEditField("Category: ", m.categoryInput) + "\n"
		s += editMutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case editStepAccount:
		s += renderEditField("Editing: ", m.selected.ID) + "\n"
		s += renderEditField("Date: ", m.dateInput) + "\n"
		s += renderEditField("Amount: ", m.amountInput) + "\n"
		s += renderEditField("Category: ", m.categoryInput) + "\n\n"
		s += renderActiveEditField("Account: ", m.accountInput) + "\n"
		s += renderEditError(m.errMessage)
		s += editMutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case editStepToAccount:
		s += renderEditField("Editing: ", m.selected.ID) + "\n"
		s += renderEditField("Date: ", m.dateInput) + "\n"
		s += renderEditField("Amount: ", m.amountInput) + "\n"
		s += renderEditField("Category: ", m.categoryInput) + "\n"
		s += renderEditField("From account: ", m.accountInput) + "\n\n"
		s += renderActiveEditField("To account: ", m.toAccountInput) + "\n"
		s += renderEditError(m.errMessage)
		s += editMutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case editStepNote:
		s += renderEditField("Editing: ", m.selected.ID) + "\n"
		s += renderEditField("Date: ", m.dateInput) + "\n"
		s += renderEditField("Amount: ", m.amountInput) + "\n"
		s += renderEditField("Category: ", m.categoryInput) + "\n"
		s += renderEditField("Account: ", m.accountInput) + "\n"
		if m.selected.Type == coininternal.TransactionTypeTransfer {
			s += renderEditField("To account: ", m.toAccountInput) + "\n"
		}
		s += "\n" + renderActiveEditField("Note: ", m.noteInput) + "\n"
		s += editMutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case editStepConfirm:
		s += renderEditField("Editing: ", m.selected.ID) + "\n"
		s += renderEditField("Date: ", m.dateInput) + "\n"
		s += renderEditField("Amount: ", m.amountInput) + "\n"
		s += renderEditField("Category: ", m.categoryInput) + "\n"
		s += renderEditField("Account: ", m.accountInput) + "\n"
		if m.selected.Type == coininternal.TransactionTypeTransfer {
			s += renderEditField("To account: ", m.toAccountInput) + "\n"
		}
		s += renderEditField("Note: ", m.noteInput) + "\n\n"
		s += editWarnStyle.Render("Save transaction changes?") + "\n\n"

		yes := "  Yes"
		no := "  No"
		if m.confirmCursor == 0 {
			yes = editFocusStyle.Render("> Yes")
		} else {
			no = editFocusStyle.Render("> No")
		}

		s += yes + "\n"
		s += no + "\n\n"
		s += editMutedStyle.Render("(use ←/→ or ↑/↓, enter to confirm, esc to go back, q to quit)") + "\n"
	case editStepDone:
		s += editMutedStyle.Render("Done") + "\n"
	}

	return s
}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit a transaction",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		transactionsPath, err := coininternal.FilePath(coininternal.TransactionsFileName)
		if err != nil {
			return err
		}

		transactionsFile, err := coininternal.LoadTransactions(transactionsPath)
		if err != nil {
			return err
		}
		if len(transactionsFile.Transactions) == 0 {
			return fmt.Errorf("no transactions available")
		}

		sortEditableTransactions(transactionsFile.Transactions)

		p := tea.NewProgram(newEditModel(transactionsFile.Transactions))
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		result := finalModel.(editModel)
		if !result.confirmed || result.selected.ID == "" {
			fmt.Println("edit cancelled")
			return nil
		}

		date := result.dateInput
		amount := result.amountInput
		category := result.categoryInput
		account := result.accountInput
		note := result.noteInput

		edits := coininternal.TransactionEdits{
			Date:     &date,
			Amount:   &amount,
			Category: &category,
			Account:  &account,
			Note:     &note,
		}
		if result.selected.Type == coininternal.TransactionTypeTransfer {
			toAccount := result.toAccountInput
			edits.ToAccount = &toAccount
		}

		tx, err := coininternal.EditTransaction(result.selected.ID, edits)
		if err != nil {
			return err
		}

		fmt.Printf("updated transaction: %s\n", tx.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}

func sortEditableTransactions(transactions []model.Transaction) {
	sort.Slice(transactions, func(i, j int) bool {
		if transactions[i].Date == transactions[j].Date {
			return transactions[i].CreatedAt > transactions[j].CreatedAt
		}
		return transactions[i].Date > transactions[j].Date
	})
}

func formatEditableTransaction(tx model.Transaction) string {
	amount := coininternal.FormatMinor(tx.AmountMinor)
	if tx.Type == coininternal.TransactionTypeExpense {
		amount = "-" + amount
	}

	details := tx.Account
	if tx.Type == coininternal.TransactionTypeTransfer {
		details = tx.Account + " -> " + tx.ToAccount
	}

	label := fmt.Sprintf("%s | %s %s | %s | %s | %s", tx.Date, amount, tx.Currency, tx.Type, details, tx.ID)
	if strings.TrimSpace(tx.Note) != "" {
		label += " | " + strings.TrimSpace(tx.Note)
	}

	return label
}

func renderEditField(label, value string) string {
	return label + editValueStyle.Render(value)
}

func renderActiveEditField(label, value string) string {
	return label + renderEditCursor(value)
}

func renderEditCursor(value string) string {
	rendered := ""
	if value != "" {
		rendered = editValueStyle.Render(value)
	}

	return rendered + editCursorStyle.Render(" ")
}

func renderEditError(message string) string {
	if message == "" {
		return ""
	}

	return editWarnStyle.Render(message) + "\n"
}

func formatEditAmountInput(amountMinor int64) string {
	whole := amountMinor / 100
	fraction := amountMinor % 100
	return fmt.Sprintf("%d.%02d", whole, fraction)
}
