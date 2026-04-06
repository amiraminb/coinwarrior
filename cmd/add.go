package cmd

import (
	"fmt"
	"strings"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type addStep int

const (
	stepType addStep = iota
	stepAmount
	stepCurrency
	stepCategorySelect
	stepCategoryInput
	stepCategoryConfirm
	stepAccountSelect
	stepAccountInput
	stepAccountConfirm
	stepDone
)

type addModel struct {
	step addStep

	cursor   int
	choices  []string
	selected string

	amountInput   string
	currencyInput string
	categoryInput string
	accountInput  string

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
}

func newAddModel(categories []string, accounts []string) addModel {
	return addModel{
		step:          stepType,
		cursor:        0,
		choices:       []string{"expense", "income"},
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
					m.step = stepCategorySelect
				}
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
					m.step = stepDone
					return m, tea.Quit
				}
				m.step = stepAccountInput
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
					m.step = stepDone
					return m, tea.Quit
				}
				m.step = stepAccountInput
			case "esc":
				m.step = stepAccountInput
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
	case stepCategorySelect:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Currency: " + m.currencyInput + "\n\n"
		s += "Select category:\n\n"
		for i, c := range m.categories {
			prefix := "  "
			if i == m.categoryCursor {
				prefix = "> "
			}
			s += prefix + c + "\n"
		}
		newOptionPrefix := "  "
		if m.categoryCursor == len(m.categories) {
			newOptionPrefix = "> "
		}
		s += newOptionPrefix + "[New category]\n"
		s += "\n(use ↑/↓ and enter, q to quit)\n"
	case stepCategoryInput:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Currency: " + m.currencyInput + "\n\n"
		s += "Enter category: " + m.categoryDraft + "\n"
		s += "(enter to continue, esc to go back, q to quit)\n"
	case stepCategoryConfirm:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Currency: " + m.currencyInput + "\n\n"
		s += "Category '" + m.pendingCategory + "' is new. Create it?\n\n"
		yesPrefix := "  "
		noPrefix := "  "
		if m.confirmCursor == 0 {
			yesPrefix = "> "
		} else {
			noPrefix = "> "
		}
		s += yesPrefix + "Yes\n"
		s += noPrefix + "No\n"
		s += "\n(use ←/→ or ↑/↓ and enter)\n"
	case stepAccountSelect:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Currency: " + m.currencyInput + "\n"
		s += "Category: " + m.categoryInput + "\n\n"
		s += "Select account:\n\n"
		for i, a := range m.accounts {
			prefix := "  "
			if i == m.accountCursor {
				prefix = "> "
			}
			s += prefix + a + "\n"
		}
		newOptionPrefix := "  "
		if m.accountCursor == len(m.accounts) {
			newOptionPrefix = "> "
		}
		s += newOptionPrefix + "[New account]\n"
		s += "\n(use ↑/↓ and enter, q to quit)\n"
	case stepAccountInput:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Currency: " + m.currencyInput + "\n"
		s += "Category: " + m.categoryInput + "\n\n"
		s += "Enter account: " + m.accountDraft + "\n"
		s += "(enter to continue, esc to go back, q to quit)\n"
	case stepAccountConfirm:
		s += "Type selected: " + m.selected + "\n"
		s += "Amount: " + m.amountInput + "\n"
		s += "Currency: " + m.currencyInput + "\n"
		s += "Category: " + m.categoryInput + "\n\n"
		s += "Account '" + m.pendingAccount + "' is new. Create it?\n\n"
		yesPrefix := "  "
		noPrefix := "  "
		if m.accountConfirm == 0 {
			yesPrefix = "> "
		} else {
			noPrefix = "> "
		}
		s += yesPrefix + "Yes\n"
		s += noPrefix + "No\n"
		s += "\n(use ←/→ or ↑/↓ and enter)\n"
	case stepDone:
		s += "Done\n"
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
		if result.selected == "" || result.amountInput == "" || result.currencyInput == "" || result.categoryInput == "" || result.accountInput == "" {
			fmt.Println("add cancelled")
			return nil
		}

		tx, err := coininternal.AddTransaction(result.selected, result.amountInput, result.currencyInput, result.categoryInput, result.accountInput)
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
