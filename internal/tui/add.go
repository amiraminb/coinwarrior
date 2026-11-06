package tui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
	tea "github.com/charmbracelet/bubbletea"
)

func containsFold(items []string, target string) bool {
	return slices.ContainsFunc(items, func(s string) bool {
		return strings.EqualFold(s, target)
	})
}

type addStep int

const (
	stepType addStep = iota
	stepAmount
	stepDate
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
	step   addStep
	preset map[addStep]bool

	cursor   int
	filter   menuFilter
	choices  []string
	selected string

	amountInput    string
	dateInput      string
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
		step:       stepType,
		cursor:     0,
		choices:    []string{model.TransactionTypeExpense, model.TransactionTypeIncome},
		dateInput:  time.Now().Format("2006-01-02"),
		categories: categories,
		accounts:   accounts,
	}
}

func (m addModel) Init() tea.Cmd {
	return nil
}

// Only confirmation prompts reserve bare "q" for quitting; lists accept it as
// filter text, and text fields accept it as input.
func (s addStep) isConfirmationStep() bool {
	switch s {
	case stepCategoryConfirm, stepAccountConfirm:
		return true
	default:
		return false
	}
}

func (m addModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	previousStep := m.step
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.step.isConfirmationStep() {
				return m, tea.Quit
			}
		}
		switch m.step {
		case stepType:
			if m.filter.update(msg, m.choices, &m.cursor) {
				return m, nil
			}
			switch msg.String() {
			case "enter":
				m.selected = m.choices[m.cursor]
				m.step = stepAmount
			case "esc":
				return m, tea.Quit
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
					m.step = stepCategorySelect
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
		case stepCategorySelect:
			if m.filter.update(msg, withNewOption(m.categories, "[New category]"), &m.categoryCursor) {
				return m, nil
			}
			switch msg.String() {
			case "enter":
				if m.categoryCursor < len(m.categories) {
					m.categoryInput = m.categories[m.categoryCursor]
					m.step = stepAccountSelect
					break
				}
				m.step = stepCategoryInput
			case "esc":
				m.step = stepDate
			}
		case stepCategoryInput:
			switch msg.String() {
			case "enter":
				draft := strings.TrimSpace(m.categoryDraft)
				if draft != "" {
					if containsFold(m.categories, draft) {
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
			if m.filter.update(msg, m.accountChoices(), &m.accountCursor) {
				return m, nil
			}
			switch msg.String() {
			case "enter":
				if m.accountCursor < len(m.accounts) {
					m.accountInput = m.accounts[m.accountCursor]
					m.createAccount = false
					if m.selected == model.TransactionTypeTransfer {
						m.step = stepTransferToAccountSelect
					} else {
						m.step = stepNote
					}
					break
				}
				if m.selected != model.TransactionTypeTransfer {
					m.step = stepAccountInput
				}
			case "esc":
				if m.selected == model.TransactionTypeTransfer {
					m.step = stepDate
				} else {
					m.step = stepCategorySelect
				}
			}
		case stepAccountInput:
			switch msg.String() {
			case "enter":
				draft := strings.TrimSpace(m.accountDraft)
				if draft != "" {
					if containsFold(m.accounts, draft) {
						m.accountInput = draft
						m.step = stepNote
						break
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
			if m.filter.update(msg, m.accounts, &m.accountCursor) {
				return m, nil
			}
			switch msg.String() {
			case "enter":
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
				if m.selected == model.TransactionTypeTransfer {
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
	if m.step != previousStep {
		m.skipPresetSteps(previousStep)
		m.filter = menuFilter{}
	}
	if m.step == stepDone {
		return m, tea.Quit
	}
	return m, nil
}

func (m addModel) accountChoices() []string {
	if m.selected == model.TransactionTypeTransfer {
		return m.accounts
	}
	return withNewOption(m.accounts, "[New account]")
}

func (m addModel) View() string {
	s := ""
	switch m.step {
	case stepType:
		s += "Select type:\n\n"
		s += m.filter.view(m.choices, m.cursor)
		s += menuHelp("cancel")
	case stepAmount:
		s += renderField("Type selected: ", m.selected) + "\n\n"
		s += renderActiveField("Enter amount: ", m.amountInput) + "\n"
		s += mutedStyle.Render("(press enter to continue, ctrl+c to quit)") + "\n"
	case stepDate:
		s += renderField("Type selected: ", m.selected) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n\n"
		s += renderActiveField("Enter date (YYYY-MM-DD): ", m.dateInput) + "\n"
		s += mutedStyle.Render("(press enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case stepCategorySelect:
		s += renderField("Type selected: ", m.selected) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += "Select category:\n\n"
		s += m.filter.view(withNewOption(m.categories, "[New category]"), m.categoryCursor)
		s += menuHelp("back")
	case stepCategoryInput:
		s += renderField("Type selected: ", m.selected) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderActiveField("Enter category: ", m.categoryDraft) + "\n"
		s += mutedStyle.Render("(enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case stepCategoryConfirm:
		s += renderField("Type selected: ", m.selected) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += warnStyle.Render("Category '"+m.pendingCategory+"' is new. Create it?") + "\n\n"
		yesPrefix := "  "
		noPrefix := "  "
		if m.confirmCursor == 0 {
			yesPrefix = focusStyle.Render("> ")
		} else {
			noPrefix = focusStyle.Render("> ")
		}
		s += yesPrefix + "Yes\n"
		s += noPrefix + "No\n"
		s += "\n" + mutedStyle.Render("(use ←/→ or ↑/↓ and enter)") + "\n"
	case stepAccountSelect:
		s += renderField("Type selected: ", m.selected) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Date: ", m.dateInput) + "\n"
		if m.selected != model.TransactionTypeTransfer {
			s += renderField("Category: ", m.categoryInput) + "\n\n"
			s += "Select account:\n\n"
		} else {
			s += "\nSelect from account:\n\n"
		}
		s += m.filter.view(m.accountChoices(), m.accountCursor)
		s += menuHelp("back")
	case stepAccountInput:
		s += renderField("Type selected: ", m.selected) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Date: ", m.dateInput) + "\n"
		s += renderField("Category: ", m.categoryInput) + "\n\n"
		s += renderActiveField("Enter account: ", m.accountDraft) + "\n"
		s += mutedStyle.Render("(enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case stepTransferToAccountSelect:
		s += renderField("Type selected: ", m.selected) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Date: ", m.dateInput) + "\n"
		s += renderField("From account: ", m.accountInput) + "\n\n"
		s += "Select to account:\n\n"
		s += m.filter.header(m.accounts)
		for _, i := range m.filter.indices(m.accounts) {
			a := m.accounts[i]
			line := "  " + a
			if strings.EqualFold(a, m.accountInput) {
				line = mutedStyle.Render("  " + a + " (same account)")
			}
			if i == m.accountCursor {
				line = focusStyle.Render("> " + a)
				if strings.EqualFold(a, m.accountInput) {
					line = warnStyle.Render("> " + a + " (same account not allowed)")
				}
			}
			s += line + "\n"
		}
		s += menuHelp("back")
	case stepAccountConfirm:
		s += renderField("Type selected: ", m.selected) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Date: ", m.dateInput) + "\n"
		s += renderField("Category: ", m.categoryInput) + "\n\n"
		s += warnStyle.Render("Account '"+m.pendingAccount+"' is new. Create it?") + "\n\n"
		yesPrefix := "  "
		noPrefix := "  "
		if m.accountConfirm == 0 {
			yesPrefix = focusStyle.Render("> ")
		} else {
			noPrefix = focusStyle.Render("> ")
		}
		s += yesPrefix + "Yes\n"
		s += noPrefix + "No\n"
		s += "\n" + mutedStyle.Render("(use ←/→ or ↑/↓ and enter)") + "\n"
	case stepNote:
		s += renderField("Type selected: ", m.selected) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Date: ", m.dateInput) + "\n"
		if m.selected != model.TransactionTypeTransfer {
			s += renderField("Category: ", m.categoryInput) + "\n"
			s += renderField("Account: ", m.accountInput) + "\n\n"
		} else {
			s += renderField("From account: ", m.accountInput) + "\n"
			s += renderField("To account: ", m.toAccountInput) + "\n"
			s += renderField("Category: ", model.TransferCategory) + "\n\n"
		}
		s += renderActiveField("Enter note (optional): ", m.noteInput) + "\n"
		s += mutedStyle.Render("(enter to save, esc to go back, ctrl+c to quit)") + "\n"
	case stepDone:
		s += mutedStyle.Render("Done") + "\n"
	}
	return s
}

func RunAddTransaction(options AddTransactionOptions) error {
	categories, err := loadCategoryChoices()
	if err != nil {
		return err
	}
	accounts, err := Svc.LoadAccountNames()
	if err != nil {
		return err
	}

	result, err := newAddModelWithOptions(categories, accounts, options)
	if err != nil {
		return err
	}
	if result.step != stepDone {
		p := tea.NewProgram(result)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		result = finalModel.(addModel)
	}
	return finishAddTransaction(result)
}

func finishAddTransaction(result addModel) error {
	if result.step != stepDone {
		fmt.Println("add cancelled")
		return nil
	}
	if result.selected == "" || result.amountInput == "" || result.accountInput == "" || result.dateInput == "" {
		fmt.Println("add cancelled")
		return nil
	}
	if result.selected != model.TransactionTypeTransfer && result.categoryInput == "" {
		fmt.Println("add cancelled")
		return nil
	}
	if result.selected == model.TransactionTypeTransfer && (result.toAccountInput == "" || strings.EqualFold(result.accountInput, result.toAccountInput)) {
		fmt.Println("add cancelled")
		return nil
	}

	// TODO: this is buggy
	proceed, err := confirmNotDuplicate(result.selected, result.amountInput, "CAD", result.dateInput, result.categoryInput)
	if err != nil {
		return err
	}
	if !proceed {
		fmt.Println("not saved; possible duplicate")
		return nil
	}

	if result.createAccount && result.selected != model.TransactionTypeTransfer {
		if _, err := Svc.AddAccount(result.accountInput, "CAD", "0"); err != nil {
			return err
		}
	}

	if result.selected != model.TransactionTypeTransfer {
		if err := Svc.AddCategory(result.categoryInput); err != nil {
			return err
		}
	}

	tx, err := Svc.AddTransaction(result.selected, result.amountInput, "CAD", result.dateInput, result.categoryInput, result.accountInput, result.toAccountInput, result.noteInput)
	if err != nil {
		return err
	}

	fmt.Printf("saved transaction: %s\n", tx.ID)
	printFutureValue(tx.AmountMinor, tx.Currency)
	return nil
}

func printFutureValue(amountMinor int64, currency string) {
	fmt.Println()
	fmt.Println(HeaderStyle.Render(fmt.Sprintf("Future Value (%d%% Annual Interest)", money.FutureValueInterestPercent)))
	for _, years := range money.FutureValueHorizons() {
		futureMinor := money.FutureValueMinor(amountMinor, years)
		fmt.Printf("  %2d years: %s %s\n", years, currency, money.Format(futureMinor))
	}
}
