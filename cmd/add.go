package cmd

import (
	"fmt"
	"strings"
	"time"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	addFocusStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	addMutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	addWarnStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
)

type addStep int

const (
	stepType addStep = iota
	stepAmount
	stepDate
	stepCurrency
	stepCategorySelect
	stepCategoryInput
	stepCategoryConfirm
	stepAccountSelect
	stepAccountInput
	stepAccountConfirm
	stepTransferToAccountSelect
	stepNote
	stepDone
)

type addModel struct {
	step addStep

	cursor   int
	choices  []string
	selected string

	amountInput    string
	dateInput      string
	currencyInput  string
	categoryInput  string
	accountInput   string
	toAccountInput string
	noteInput      string

	categories      []string
	categoryCursor  int
	categoryDraft   string
	pendingCategory string
	confirmCursor   int

	accounts       []string
	accountCursor  int
	accountDraft   string
	pendingAccount string
	accountConfirm int
	createAccount  bool
}

func newAddModel(categories []string, accounts []string) addModel {
	return addModel{
		step:          stepType,
		cursor:        0,
		choices:       []string{coininternal.TransactionTypeExpense, coininternal.TransactionTypeIncome, coininternal.TransactionTypeTransfer},
		dateInput:     time.Now().Format("2006-01-02"),
		currencyInput: "CAD",
		categories:    categories,
		accounts:      accounts,
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
					m.step = stepDate
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
		case stepDate:
			switch msg.String() {
			case "enter":
				if _, err := time.Parse("2006-01-02", strings.TrimSpace(m.dateInput)); err == nil {
					m.step = stepCurrency
				}
			case "esc":
				m.step = stepAmount
			case "backspace":
				if len(m.dateInput) > 0 {
					m.dateInput = m.dateInput[:len(m.dateInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					ch := msg.String()
					if (ch >= "0" && ch <= "9") || ch == "-" {
						m.dateInput += ch
					}
				}
			}
		case stepCurrency:
			switch msg.String() {
			case "enter":
				if m.currencyInput != "" {
					if m.selected == coininternal.TransactionTypeTransfer {
						m.step = stepAccountSelect
					} else {
						m.step = stepCategorySelect
					}
				}
			case "esc":
				m.step = stepDate
			case "backspace":
				if len(m.currencyInput) > 0 {
					m.currencyInput = m.currencyInput[:len(m.currencyInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					ch := strings.ToUpper(msg.String())
					if len(m.currencyInput) < 3 && ((ch >= "A" && ch <= "Z") || (ch >= "a" && ch <= "z")) {
						m.currencyInput += ch
					}
				}
			}
		case stepCategorySelect:
			maxCursor := len(m.categories)
			switch msg.String() {
			case "up", "k":
				if m.categoryCursor > 0 {
					m.categoryCursor--
				}
			case "down", "j":
				if m.categoryCursor < maxCursor {
					m.categoryCursor++
				}
			case "enter":
				if m.categoryCursor < len(m.categories) {
					m.categoryInput = m.categories[m.categoryCursor]
					m.step = stepAccountSelect
					break
				}
				m.step = stepCategoryInput
			}
		case stepCategoryInput:
			switch msg.String() {
			case "enter":
				draft := strings.TrimSpace(m.categoryDraft)
				if draft != "" {
					if coininternal.CategoryExists(m.categories, draft) {
						m.categoryInput = draft
						m.step = stepAccountSelect
						break
					}
					m.pendingCategory = draft
					m.confirmCursor = 0
					m.step = stepCategoryConfirm
				}
			case "esc":
				m.step = stepCategorySelect
			case "backspace":
				if len(m.categoryDraft) > 0 {
					m.categoryDraft = m.categoryDraft[:len(m.categoryDraft)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.categoryDraft += msg.String()
				}
			}
		case stepCategoryConfirm:
			switch msg.String() {
			case "left", "h", "up", "k":
				m.confirmCursor = 0
			case "right", "l", "down", "j":
				m.confirmCursor = 1
			case "enter":
				if m.confirmCursor == 0 {
					m.categoryInput = m.pendingCategory
					m.step = stepAccountSelect
					break
				}
				m.step = stepCategoryInput
			case "esc":
				m.step = stepCategoryInput
			}
		case stepAccountSelect:
			maxCursor := len(m.accounts)
			if m.selected == coininternal.TransactionTypeTransfer {
				if len(m.accounts) == 0 {
					maxCursor = 0
				} else {
					maxCursor = len(m.accounts) - 1
				}
			}
			switch msg.String() {
			case "up", "k":
				if m.accountCursor > 0 {
					m.accountCursor--
				}
			case "down", "j":
				if m.accountCursor < maxCursor {
					m.accountCursor++
				}
			case "enter":
				if m.accountCursor < len(m.accounts) {
					m.accountInput = m.accounts[m.accountCursor]
					m.createAccount = false
					if m.selected == coininternal.TransactionTypeTransfer {
						m.step = stepTransferToAccountSelect
					} else {
						m.step = stepNote
					}
					break
				}
				if m.selected != coininternal.TransactionTypeTransfer {
					m.step = stepAccountInput
				}
			}
		case stepAccountInput:
			switch msg.String() {
			case "enter":
				draft := strings.TrimSpace(m.accountDraft)
				if draft != "" {
					if coininternal.AccountExists(m.accounts, draft) {
						m.accountInput = draft
						m.step = stepDone
						return m, tea.Quit
					}
					m.pendingAccount = draft
					m.accountConfirm = 0
					m.step = stepAccountConfirm
				}
			case "esc":
				m.step = stepAccountSelect
			case "backspace":
				if len(m.accountDraft) > 0 {
					m.accountDraft = m.accountDraft[:len(m.accountDraft)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.accountDraft += msg.String()
				}
			}
		case stepAccountConfirm:
			switch msg.String() {
			case "left", "h", "up", "k":
				m.accountConfirm = 0
			case "right", "l", "down", "j":
				m.accountConfirm = 1
			case "enter":
				if m.accountConfirm == 0 {
					m.accountInput = m.pendingAccount
					m.createAccount = true
					m.step = stepNote
					break
				}
				m.step = stepAccountInput
			case "esc":
				m.step = stepAccountInput
			}
		case stepTransferToAccountSelect:
			switch msg.String() {
			case "up", "k":
				if m.accountCursor > 0 {
					m.accountCursor--
				}
			case "down", "j":
				if m.accountCursor < len(m.accounts)-1 {
					m.accountCursor++
				}
			case "enter":
				if len(m.accounts) == 0 {
					break
				}
				candidate := m.accounts[m.accountCursor]
				if strings.EqualFold(candidate, m.accountInput) {
					break
				}
				m.toAccountInput = candidate
				m.step = stepNote
			case "esc":
				m.step = stepAccountSelect
			}
		case stepNote:
			switch msg.String() {
			case "enter":
				m.step = stepDone
				return m, tea.Quit
			case "esc":
				if m.selected == coininternal.TransactionTypeTransfer {
					m.step = stepTransferToAccountSelect
				} else {
					m.step = stepAccountSelect
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
		}
	}
	return m, nil
}

func (m addModel) View() string {
	s := ""
	switch m.step {
	case stepType:
		s += "Select type:\n\n"
		for i, c := range m.choices {
			line := "  " + c
			if i == m.cursor {
				line = addFocusStyle.Render("> " + c)
			}
			s += line + "\n"
		}
		s += "\n" + addMutedStyle.Render("(use ↑/↓ and enter, q to quit)") + "\n"
	case stepAmount:
		s += "Type selected: " + m.selected + "\n\n"
		s += "Enter amount: " + m.amountInput + "\n"
		s += addMutedStyle.Render("(press enter to continue, q to quit)") + "\n"
	case stepDate:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n\n"
		s += "Enter date (YYYY-MM-DD): " + m.dateInput + "\n"
		s += addMutedStyle.Render("(press enter to continue, esc to go back, q to quit)") + "\n"
	case stepCurrency:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n\n"
		s += "Enter currency: " + m.currencyInput + "\n"
		s += addMutedStyle.Render("(press enter to continue, esc to go back, q to quit)") + "\n"
	case stepCategorySelect:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Currency: " + m.currencyInput + "\n\n"
		s += "Select category:\n\n"
		for i, c := range m.categories {
			line := "  " + c
			if i == m.categoryCursor {
				line = addFocusStyle.Render("> " + c)
			}
			s += line + "\n"
		}
		newOptionLine := "  [New category]"
		if m.categoryCursor == len(m.categories) {
			newOptionLine = addFocusStyle.Render("> [New category]")
		}
		s += newOptionLine + "\n"
		s += "\n" + addMutedStyle.Render("(use ↑/↓ and enter, q to quit)") + "\n"
	case stepCategoryInput:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Currency: " + m.currencyInput + "\n\n"
		s += "Enter category: " + m.categoryDraft + "\n"
		s += addMutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case stepCategoryConfirm:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Currency: " + m.currencyInput + "\n\n"
		s += addWarnStyle.Render("Category '"+m.pendingCategory+"' is new. Create it?") + "\n\n"
		yesPrefix := "  "
		noPrefix := "  "
		if m.confirmCursor == 0 {
			yesPrefix = addFocusStyle.Render("> ")
		} else {
			noPrefix = addFocusStyle.Render("> ")
		}
		s += yesPrefix + "Yes\n"
		s += noPrefix + "No\n"
		s += "\n" + addMutedStyle.Render("(use ←/→ or ↑/↓ and enter)") + "\n"
	case stepAccountSelect:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Date: " + m.dateInput + "\n"
		s += "Currency: " + m.currencyInput + "\n"
		if m.selected != coininternal.TransactionTypeTransfer {
			s += "Category: " + m.categoryInput + "\n\n"
			s += "Select account:\n\n"
		} else {
			s += "\nSelect from account:\n\n"
		}
		for i, a := range m.accounts {
			line := "  " + a
			if i == m.accountCursor {
				line = addFocusStyle.Render("> " + a)
			}
			s += line + "\n"
		}
		if m.selected != coininternal.TransactionTypeTransfer {
			newOptionLine := "  [New account]"
			if m.accountCursor == len(m.accounts) {
				newOptionLine = addFocusStyle.Render("> [New account]")
			}
			s += newOptionLine + "\n"
		}
		s += "\n" + addMutedStyle.Render("(use ↑/↓ and enter, q to quit)") + "\n"
	case stepAccountInput:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Date: " + m.dateInput + "\n"
		s += "Currency: " + m.currencyInput + "\n"
		s += "Category: " + m.categoryInput + "\n\n"
		s += "Enter account: " + m.accountDraft + "\n"
		s += addMutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case stepTransferToAccountSelect:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Date: " + m.dateInput + "\n"
		s += "Currency: " + m.currencyInput + "\n"
		s += "From account: " + m.accountInput + "\n\n"
		s += "Select to account:\n\n"
		for i, a := range m.accounts {
			line := "  " + a
			if strings.EqualFold(a, m.accountInput) {
				line = addMutedStyle.Render("  " + a + " (same account)")
			}
			if i == m.accountCursor {
				line = addFocusStyle.Render("> " + a)
				if strings.EqualFold(a, m.accountInput) {
					line = addWarnStyle.Render("> " + a + " (same account not allowed)")
				}
			}
			s += line + "\n"
		}
		s += "\n" + addMutedStyle.Render("(use ↑/↓ and enter, esc to go back, q to quit)") + "\n"
	case stepAccountConfirm:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Date: " + m.dateInput + "\n"
		s += "Currency: " + m.currencyInput + "\n"
		s += "Category: " + m.categoryInput + "\n\n"
		s += addWarnStyle.Render("Account '"+m.pendingAccount+"' is new. Create it?") + "\n\n"
		yesPrefix := "  "
		noPrefix := "  "
		if m.accountConfirm == 0 {
			yesPrefix = addFocusStyle.Render("> ")
		} else {
			noPrefix = addFocusStyle.Render("> ")
		}
		s += yesPrefix + "Yes\n"
		s += noPrefix + "No\n"
		s += "\n" + addMutedStyle.Render("(use ←/→ or ↑/↓ and enter)") + "\n"
	case stepNote:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Date: " + m.dateInput + "\n"
		s += "Currency: " + m.currencyInput + "\n"
		if m.selected != coininternal.TransactionTypeTransfer {
			s += "Category: " + m.categoryInput + "\n"
			s += "Account: " + m.accountInput + "\n\n"
		} else {
			s += "From account: " + m.accountInput + "\n"
			s += "To account: " + m.toAccountInput + "\n"
			s += "Category: Transfer\n\n"
		}
		s += "Enter note (optional): " + m.noteInput + "\n"
		s += addMutedStyle.Render("(enter to save, esc to go back, q to quit)") + "\n"
	case stepDone:
		s += addMutedStyle.Render("Done") + "\n"
	}
	return s
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a transaction",
	RunE: func(cmd *cobra.Command, args []string) error {
		categories, err := coininternal.LoadCategories()
		if err != nil {
			return err
		}
		accounts, err := coininternal.LoadAccounts()
		if err != nil {
			return err
		}

		m := newAddModel(categories, accounts)
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		result := finalModel.(addModel)
		if result.selected == "" || result.amountInput == "" || result.currencyInput == "" || result.accountInput == "" || result.dateInput == "" {
			fmt.Println("add cancelled")
			return nil
		}
		if result.selected != coininternal.TransactionTypeTransfer && result.categoryInput == "" {
			fmt.Println("add cancelled")
			return nil
		}
		if result.selected == coininternal.TransactionTypeTransfer && (result.toAccountInput == "" || strings.EqualFold(result.accountInput, result.toAccountInput)) {
			fmt.Println("add cancelled")
			return nil
		}

		if result.createAccount && result.selected != coininternal.TransactionTypeTransfer {
			if _, err := coininternal.AddAccount(result.accountInput, result.currencyInput, "0"); err != nil {
				return err
			}
		}

		if result.selected != coininternal.TransactionTypeTransfer {
			if err := coininternal.AddCategory(result.categoryInput); err != nil {
				return err
			}
		}

		tx, err := coininternal.AddTransaction(result.selected, result.amountInput, result.currencyInput, result.dateInput, result.categoryInput, result.accountInput, result.toAccountInput, result.noteInput)
		if err != nil {
			return err
		}

		fmt.Printf("saved transaction: %s\n", tx.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
