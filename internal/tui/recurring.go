package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
	"github.com/amiraminb/coinwarrior/internal/service"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type recurringAction string

const (
	recurringActionList     recurringAction = "list"
	recurringActionAdd      recurringAction = "add"
	recurringActionEdit     recurringAction = "edit"
	recurringActionDelete   recurringAction = "delete"
	recurringActionGenerate recurringAction = "generate"
	recurringActionQuit     recurringAction = "quit"
)

func RunRecurringAction() error {
	items := []selectionItem[recurringAction]{
		{label: "List rules", value: recurringActionList},
		{label: "Add rule", value: recurringActionAdd},
		{label: "Edit rule", value: recurringActionEdit},
		{label: "Delete rule", value: recurringActionDelete},
		{label: "Generate due transactions", value: recurringActionGenerate},
		{label: "Quit", value: recurringActionQuit},
	}
	action, ok, err := runSelection("Recurring Transactions", "Choose an action:", items)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	switch action {
	case recurringActionList:
		return runRecurringList()
	case recurringActionAdd:
		_, err := runRecurringAdd()
		return err
	case recurringActionEdit:
		_, err := runRecurringEdit()
		return err
	case recurringActionDelete:
		_, err := runRecurringDelete()
		return err
	case recurringActionGenerate:
		return runRecurringGenerate()
	case recurringActionQuit:
		return nil
	default:
		fmt.Println("recurring cancelled")
		return nil
	}
}

func runRecurringList() error {
	rules, err := Svc.LoadRecurringRules()
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		fmt.Println("no recurring rules")
		return nil
	}

	rows := make([]table.Row, 0, len(rules))
	for _, rule := range rules {
		account := rule.Account
		if rule.Type == model.TransactionTypeTransfer {
			account = rule.Account + " -> " + rule.ToAccount
		}
		end := rule.EndDate
		if end == "" {
			end = "-"
		}
		last := rule.LastGeneratedMonth
		if last == "" {
			last = "-"
		}
		rows = append(rows, table.Row{
			rule.ID,
			rule.Type,
			money.Format(rule.AmountMinor),
			rule.Currency,
			rule.Category,
			account,
			strconv.Itoa(rule.DayOfMonth),
			rule.StartDate,
			end,
			last,
		})
	}

	RenderTable(
		[]table.Column{
			{Title: "ID", Width: 24},
			{Title: "TYPE", Width: 8},
			{Title: "AMOUNT", Width: 12},
			{Title: "CUR", Width: 5},
			{Title: "CATEGORY", Width: 14},
			{Title: "ACCOUNT", Width: 22},
			{Title: "DAY", Width: 4},
			{Title: "START", Width: 10},
			{Title: "END", Width: 10},
			{Title: "LAST GEN", Width: 10},
		},
		rows,
	)
	return nil
}

type recurringField int

const (
	recurringFieldType recurringField = iota
	recurringFieldAmount
	recurringFieldCurrency
	recurringFieldCategory
	recurringFieldAccount
	recurringFieldToAccount
	recurringFieldDayOfMonth
	recurringFieldStartDate
	recurringFieldEndDate
	recurringFieldNote
	recurringFieldConfirm
	recurringFieldDone
)

type recurringFormModel struct {
	step            recurringField
	typeChoices     []string
	typeCursor      int
	amountInput     string
	currencyInput   string
	categories      []string
	categoryCursor  int
	accounts        []string
	accountCursor   int
	toAccountCursor int
	dayInput        string
	startInput      string
	endInput        string
	noteInput       string
	confirmCursor   int
	confirmed       bool
	errMessage      string
	editingID       string // empty for add
}

func newRecurringAddFormModel(categories, accounts []string) recurringFormModel {
	return recurringFormModel{
		step:          recurringFieldType,
		typeChoices:   []string{model.TransactionTypeExpense, model.TransactionTypeIncome, model.TransactionTypeTransfer},
		currencyInput: "CAD",
		startInput:    time.Now().Format("2006-01-02"),
		dayInput:      "1",
		categories:    categories,
		accounts:      accounts,
	}
}

func newRecurringEditFormModel(rule model.RecurringRule, categories, accounts []string) recurringFormModel {
	typeIdx := 0
	choices := []string{model.TransactionTypeExpense, model.TransactionTypeIncome, model.TransactionTypeTransfer}
	for i, c := range choices {
		if c == rule.Type {
			typeIdx = i
			break
		}
	}
	categoryCursor := indexOfFold(categories, rule.Category)
	if categoryCursor < 0 {
		categoryCursor = 0
	}
	accountCursor := indexOfFold(accounts, rule.Account)
	if accountCursor < 0 {
		accountCursor = 0
	}
	toAccountCursor := indexOfFold(accounts, rule.ToAccount)
	if toAccountCursor < 0 {
		toAccountCursor = 0
	}
	return recurringFormModel{
		step:            recurringFieldType,
		typeChoices:     choices,
		typeCursor:      typeIdx,
		amountInput:     formatRuleAmountInput(rule.AmountMinor),
		currencyInput:   rule.Currency,
		categories:      categories,
		categoryCursor:  categoryCursor,
		accounts:        accounts,
		accountCursor:   accountCursor,
		toAccountCursor: toAccountCursor,
		dayInput:        strconv.Itoa(rule.DayOfMonth),
		startInput:      rule.StartDate,
		endInput:        rule.EndDate,
		noteInput:       rule.Note,
		editingID:       rule.ID,
	}
}

func indexOfFold(items []string, target string) int {
	for i, v := range items {
		if strings.EqualFold(v, target) {
			return i
		}
	}
	return -1
}

func (m recurringFormModel) selectedCategory() string {
	if m.categoryCursor < 0 || m.categoryCursor >= len(m.categories) {
		return ""
	}
	return m.categories[m.categoryCursor]
}

func (m recurringFormModel) selectedAccount() string {
	if m.accountCursor < 0 || m.accountCursor >= len(m.accounts) {
		return ""
	}
	return m.accounts[m.accountCursor]
}

func (m recurringFormModel) selectedToAccount() string {
	if m.toAccountCursor < 0 || m.toAccountCursor >= len(m.accounts) {
		return ""
	}
	return m.accounts[m.toAccountCursor]
}

func renderSelectionList(items []string, cursor int) string {
	out := ""
	for i, item := range items {
		line := "  " + item
		if i == cursor {
			line = focusStyle.Render("> " + item)
		}
		out += line + "\n"
	}
	return out
}

func formatRuleAmountInput(amountMinor int64) string {
	whole := amountMinor / 100
	fraction := amountMinor % 100
	return fmt.Sprintf("%d.%02d", whole, fraction)
}

func (m recurringFormModel) Init() tea.Cmd { return nil }

// isSelectionStep reports whether the step is a menu (type/category/account
// list or Yes/No confirm) rather than free-text entry. On menu steps a bare "q"
// quits; on text steps it must be typed into the field.
func (s recurringField) isSelectionStep() bool {
	switch s {
	case recurringFieldType, recurringFieldCategory, recurringFieldAccount,
		recurringFieldToAccount, recurringFieldConfirm:
		return true
	default:
		return false
	}
}

func (m recurringFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.step.isSelectionStep() {
				return m, tea.Quit
			}
		}

		switch m.step {
		case recurringFieldType:
			switch msg.String() {
			case "up", "k":
				if m.typeCursor > 0 {
					m.typeCursor--
				}
			case "down", "j":
				if m.typeCursor < len(m.typeChoices)-1 {
					m.typeCursor++
				}
			case "enter":
				m.step = recurringFieldAmount
			case "esc":
				return m, tea.Quit
			}
		case recurringFieldAmount:
			switch msg.String() {
			case "enter":
				if _, err := money.Parse(m.amountInput); err != nil {
					m.errMessage = err.Error()
					break
				}
				m.errMessage = ""
				m.step = recurringFieldCurrency
			case "esc":
				m.step = recurringFieldType
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
					}
				}
			}
		case recurringFieldCurrency:
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.currencyInput) == "" {
					m.errMessage = "currency is required"
					break
				}
				m.errMessage = ""
				m.step = recurringFieldCategory
			case "esc":
				m.step = recurringFieldAmount
			case "backspace":
				if len(m.currencyInput) > 0 {
					m.currencyInput = m.currencyInput[:len(m.currencyInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					ch := strings.ToUpper(msg.String())
					if len(m.currencyInput) < 3 && ch >= "A" && ch <= "Z" {
						m.currencyInput += ch
					}
				}
			}
		case recurringFieldCategory:
			switch msg.String() {
			case "up", "k":
				if m.categoryCursor > 0 {
					m.categoryCursor--
				}
			case "down", "j":
				if m.categoryCursor < len(m.categories)-1 {
					m.categoryCursor++
				}
			case "enter":
				m.errMessage = ""
				m.step = recurringFieldAccount
			case "esc":
				m.step = recurringFieldCurrency
			}
		case recurringFieldAccount:
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
				m.errMessage = ""
				if m.typeChoices[m.typeCursor] == model.TransactionTypeTransfer {
					if m.toAccountCursor == m.accountCursor {
						if m.toAccountCursor < len(m.accounts)-1 {
							m.toAccountCursor++
						} else if m.toAccountCursor > 0 {
							m.toAccountCursor--
						}
					}
					m.step = recurringFieldToAccount
				} else {
					m.step = recurringFieldDayOfMonth
				}
			case "esc":
				m.step = recurringFieldCategory
			}
		case recurringFieldToAccount:
			switch msg.String() {
			case "up", "k":
				if m.toAccountCursor > 0 {
					m.toAccountCursor--
				}
			case "down", "j":
				if m.toAccountCursor < len(m.accounts)-1 {
					m.toAccountCursor++
				}
			case "enter":
				if m.toAccountCursor == m.accountCursor {
					m.errMessage = "source and destination accounts must be different"
					break
				}
				m.errMessage = ""
				m.step = recurringFieldDayOfMonth
			case "esc":
				m.step = recurringFieldAccount
			}
		case recurringFieldDayOfMonth:
			switch msg.String() {
			case "enter":
				day, err := strconv.Atoi(strings.TrimSpace(m.dayInput))
				if err != nil || day < 1 || day > 28 {
					m.errMessage = "day of month must be between 1 and 28"
					break
				}
				m.errMessage = ""
				m.step = recurringFieldStartDate
			case "esc":
				if m.typeChoices[m.typeCursor] == model.TransactionTypeTransfer {
					m.step = recurringFieldToAccount
				} else {
					m.step = recurringFieldAccount
				}
			case "backspace":
				m.errMessage = ""
				if len(m.dayInput) > 0 {
					m.dayInput = m.dayInput[:len(m.dayInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					ch := msg.String()
					if ch >= "0" && ch <= "9" {
						m.dayInput += ch
					}
				}
			}
		case recurringFieldStartDate:
			switch msg.String() {
			case "enter":
				if _, err := time.Parse("2006-01-02", strings.TrimSpace(m.startInput)); err != nil {
					m.errMessage = "invalid date format (YYYY-MM-DD)"
					break
				}
				m.errMessage = ""
				m.step = recurringFieldEndDate
			case "esc":
				m.step = recurringFieldDayOfMonth
			case "backspace":
				m.errMessage = ""
				if len(m.startInput) > 0 {
					m.startInput = m.startInput[:len(m.startInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					ch := msg.String()
					if (ch >= "0" && ch <= "9") || ch == "-" {
						m.startInput += ch
					}
				}
			}
		case recurringFieldEndDate:
			switch msg.String() {
			case "enter":
				if e := strings.TrimSpace(m.endInput); e != "" {
					if _, err := time.Parse("2006-01-02", e); err != nil {
						m.errMessage = "invalid date format (YYYY-MM-DD), or leave blank"
						break
					}
				}
				m.errMessage = ""
				m.step = recurringFieldNote
			case "esc":
				m.step = recurringFieldStartDate
			case "backspace":
				m.errMessage = ""
				if len(m.endInput) > 0 {
					m.endInput = m.endInput[:len(m.endInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					ch := msg.String()
					if (ch >= "0" && ch <= "9") || ch == "-" {
						m.endInput += ch
					}
				}
			}
		case recurringFieldNote:
			switch msg.String() {
			case "enter":
				m.confirmCursor = 0
				m.step = recurringFieldConfirm
			case "esc":
				m.step = recurringFieldEndDate
			case "backspace":
				if len(m.noteInput) > 0 {
					m.noteInput = m.noteInput[:len(m.noteInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.noteInput += msg.String()
				}
			}
		case recurringFieldConfirm:
			switch msg.String() {
			case "left", "h", "up", "k":
				m.confirmCursor = 0
			case "right", "l", "down", "j":
				m.confirmCursor = 1
			case "enter":
				if m.confirmCursor == 0 {
					m.confirmed = true
					m.step = recurringFieldDone
					return m, tea.Quit
				}
				m.step = recurringFieldNote
			case "esc":
				m.step = recurringFieldNote
			}
		}
	}
	return m, nil
}

func (m recurringFormModel) View() string {
	title := "Add Recurring Rule"
	if m.editingID != "" {
		title = "Edit Recurring Rule"
	}
	s := title + "\n\n"

	switch m.step {
	case recurringFieldType:
		s += "Select type:\n\n"
		for i, c := range m.typeChoices {
			line := "  " + c
			if i == m.typeCursor {
				line = focusStyle.Render("> " + c)
			}
			s += line + "\n"
		}
		s += "\n" + mutedStyle.Render("(↑/↓ enter, esc to cancel, q to quit)") + "\n"
	case recurringFieldAmount:
		s += renderField("Type: ", m.typeChoices[m.typeCursor]) + "\n\n"
		s += renderActiveField("Amount: ", m.amountInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case recurringFieldCurrency:
		s += renderField("Type: ", m.typeChoices[m.typeCursor]) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n\n"
		s += renderActiveField("Currency: ", m.currencyInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case recurringFieldCategory:
		s += renderField("Type: ", m.typeChoices[m.typeCursor]) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Currency: ", m.currencyInput) + "\n\n"
		s += "Select category:\n\n"
		s += renderSelectionList(m.categories, m.categoryCursor)
		s += "\n" + mutedStyle.Render("(↑/↓ enter, esc to go back, q to quit)") + "\n"
	case recurringFieldAccount:
		s += renderField("Type: ", m.typeChoices[m.typeCursor]) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Currency: ", m.currencyInput) + "\n"
		s += renderField("Category: ", m.selectedCategory()) + "\n\n"
		header := "Select account:"
		if m.typeChoices[m.typeCursor] == model.TransactionTypeTransfer {
			header = "Select from account:"
		}
		s += header + "\n\n"
		s += renderSelectionList(m.accounts, m.accountCursor)
		s += renderError(m.errMessage)
		s += "\n" + mutedStyle.Render("(↑/↓ enter, esc to go back, q to quit)") + "\n"
	case recurringFieldToAccount:
		s += renderField("Type: ", m.typeChoices[m.typeCursor]) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("From account: ", m.selectedAccount()) + "\n\n"
		s += "Select to account:\n\n"
		out := ""
		for i, a := range m.accounts {
			line := "  " + a
			if i == m.accountCursor {
				line = mutedStyle.Render("  " + a + " (same as from)")
			}
			if i == m.toAccountCursor {
				if i == m.accountCursor {
					line = warnStyle.Render("> " + a + " (same as from, choose another)")
				} else {
					line = focusStyle.Render("> " + a)
				}
			}
			out += line + "\n"
		}
		s += out
		s += renderError(m.errMessage)
		s += "\n" + mutedStyle.Render("(↑/↓ enter, esc to go back, q to quit)") + "\n"
	case recurringFieldDayOfMonth:
		s += renderField("Type: ", m.typeChoices[m.typeCursor]) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Account: ", m.selectedAccount()) + "\n\n"
		s += renderActiveField("Day of month (1-28): ", m.dayInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case recurringFieldStartDate:
		s += renderField("Day of month: ", m.dayInput) + "\n\n"
		s += renderActiveField("Start date (YYYY-MM-DD): ", m.startInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case recurringFieldEndDate:
		s += renderField("Start date: ", m.startInput) + "\n\n"
		s += renderActiveField("End date (YYYY-MM-DD, blank for none): ", m.endInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case recurringFieldNote:
		s += renderActiveField("Note (optional): ", m.noteInput) + "\n"
		s += mutedStyle.Render("(enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case recurringFieldConfirm:
		s += renderField("Type: ", m.typeChoices[m.typeCursor]) + "\n"
		s += renderField("Amount: ", m.amountInput) + " " + m.currencyInput + "\n"
		s += renderField("Category: ", m.selectedCategory()) + "\n"
		if m.typeChoices[m.typeCursor] == model.TransactionTypeTransfer {
			s += renderField("From -> To: ", m.selectedAccount()+" -> "+m.selectedToAccount()) + "\n"
		} else {
			s += renderField("Account: ", m.selectedAccount()) + "\n"
		}
		s += renderField("Day of month: ", m.dayInput) + "\n"
		s += renderField("Start: ", m.startInput) + "\n"
		end := m.endInput
		if end == "" {
			end = "(none)"
		}
		s += renderField("End: ", end) + "\n"
		if m.noteInput != "" {
			s += renderField("Note: ", m.noteInput) + "\n"
		}
		s += "\n" + warnStyle.Render("Save recurring rule?") + "\n\n"
		s += renderYesNo(m.confirmCursor == 0) + "\n"
		s += mutedStyle.Render("(use ←/→ or ↑/↓, enter to confirm, esc to go back, q to quit)") + "\n"
	case recurringFieldDone:
		s += mutedStyle.Render("Done") + "\n"
	}
	return s
}

func loadRecurringFormDeps() (categories, accounts []string, err error) {
	categories, err = Svc.LoadCategories()
	if err != nil {
		return nil, nil, err
	}
	if len(categories) == 0 {
		return nil, nil, fmt.Errorf("no categories available; create one with 'coinw add'")
	}
	accounts, err = Svc.LoadAccountNames()
	if err != nil {
		return nil, nil, err
	}
	if len(accounts) == 0 {
		return nil, nil, fmt.Errorf("no accounts available; create one with 'coinw account'")
	}
	return categories, accounts, nil
}

func runRecurringAdd() (bool, error) {
	categories, accounts, err := loadRecurringFormDeps()
	if err != nil {
		return false, err
	}

	p := tea.NewProgram(newRecurringAddFormModel(categories, accounts))
	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}
	result := finalModel.(recurringFormModel)
	if !result.confirmed {
		fmt.Println("recurring add cancelled")
		return false, nil
	}

	day, _ := strconv.Atoi(strings.TrimSpace(result.dayInput))
	input := service.RecurringRuleInput{
		Type:        result.typeChoices[result.typeCursor],
		AmountInput: result.amountInput,
		Currency:    result.currencyInput,
		Category:    result.selectedCategory(),
		Account:     result.selectedAccount(),
		ToAccount:   result.selectedToAccount(),
		Note:        result.noteInput,
		DayOfMonth:  day,
		StartDate:   result.startInput,
		EndDate:     result.endInput,
	}
	if input.Type != model.TransactionTypeTransfer {
		input.ToAccount = ""
	}
	rule, err := Svc.AddRecurringRule(input)
	if err != nil {
		return false, err
	}
	fmt.Printf("recurring rule created: %s\n", rule.ID)
	return true, nil
}

func runRecurringEdit() (bool, error) {
	rule, ok, err := pickRecurringRule("Edit Recurring Rule")
	if err != nil {
		return false, err
	}
	if !ok {
		fmt.Println("edit cancelled")
		return false, nil
	}

	categories, accounts, err := loadRecurringFormDeps()
	if err != nil {
		return false, err
	}

	p := tea.NewProgram(newRecurringEditFormModel(rule, categories, accounts))
	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}
	result := finalModel.(recurringFormModel)
	if !result.confirmed {
		fmt.Println("edit cancelled")
		return false, nil
	}

	day, _ := strconv.Atoi(strings.TrimSpace(result.dayInput))
	txType := result.typeChoices[result.typeCursor]
	currency := result.currencyInput
	category := result.selectedCategory()
	account := result.selectedAccount()
	toAccount := ""
	if txType == model.TransactionTypeTransfer {
		toAccount = result.selectedToAccount()
	}
	note := result.noteInput
	start := result.startInput
	end := result.endInput
	edits := service.RecurringRuleEdits{
		Type:       &txType,
		Currency:   &currency,
		Category:   &category,
		Account:    &account,
		ToAccount:  &toAccount,
		Note:       &note,
		DayOfMonth: &day,
		StartDate:  &start,
		EndDate:    &end,
	}
	amount := result.amountInput
	edits.AmountInput = &amount

	updated, err := Svc.EditRecurringRule(rule.ID, edits)
	if err != nil {
		return false, err
	}
	fmt.Printf("recurring rule updated: %s\n", updated.ID)
	return true, nil
}

func runRecurringDelete() (bool, error) {
	rule, ok, err := pickRecurringRule("Delete Recurring Rule")
	if err != nil {
		return false, err
	}
	if !ok {
		fmt.Println("delete cancelled")
		return false, nil
	}

	confirmed, err := RunConfirmPrompt("Delete this recurring rule?\n" + formatRecurringRuleSummary(rule))
	if err != nil {
		return false, err
	}
	if !confirmed {
		fmt.Println("delete cancelled")
		return false, nil
	}

	deleted, err := Svc.DeleteRecurringRule(rule.ID)
	if err != nil {
		return false, err
	}
	fmt.Printf("recurring rule deleted: %s\n", deleted.ID)
	return true, nil
}

func runRecurringGenerate() error {
	result, err := Svc.GenerateDueTransactions(time.Now())
	if err != nil {
		return err
	}
	if len(result.Generated) == 0 {
		fmt.Println("no due transactions")
		return nil
	}
	fmt.Printf("generated %d transaction(s) from %d rule(s)\n", len(result.Generated), len(result.RuleCounts))
	for _, tx := range result.Generated {
		fmt.Printf("  %s %s %s %s %s %s\n", tx.ID, tx.Date, tx.Type, money.Format(tx.AmountMinor), tx.Currency, tx.Category)
	}
	return nil
}

func pickRecurringRule(title string) (model.RecurringRule, bool, error) {
	rules, err := Svc.LoadRecurringRules()
	if err != nil {
		return model.RecurringRule{}, false, err
	}
	if len(rules) == 0 {
		return model.RecurringRule{}, false, fmt.Errorf("no recurring rules available")
	}

	items := make([]selectionItem[string], 0, len(rules))
	for _, rule := range rules {
		items = append(items, selectionItem[string]{
			label: formatRecurringRuleSummary(rule),
			value: rule.ID,
		})
	}
	id, ok, err := runSelection(title, "Select rule:", items)
	if err != nil {
		return model.RecurringRule{}, false, err
	}
	if !ok {
		return model.RecurringRule{}, false, nil
	}
	for _, rule := range rules {
		if rule.ID == id {
			return rule, true, nil
		}
	}
	return model.RecurringRule{}, false, fmt.Errorf("rule not found")
}

func formatRecurringRuleSummary(rule model.RecurringRule) string {
	account := rule.Account
	if rule.Type == model.TransactionTypeTransfer {
		account = rule.Account + " -> " + rule.ToAccount
	}
	return fmt.Sprintf("%s | %s %s | %s | %s | day %d | from %s",
		rule.Type,
		money.Format(rule.AmountMinor),
		rule.Currency,
		rule.Category,
		account,
		rule.DayOfMonth,
		rule.StartDate,
	)
}
