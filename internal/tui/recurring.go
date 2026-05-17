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
	step           recurringField
	typeChoices    []string
	typeCursor     int
	amountInput    string
	currencyInput  string
	categoryInput  string
	accountInput   string
	toAccountInput string
	dayInput       string
	startInput     string
	endInput       string
	noteInput      string
	confirmCursor  int
	confirmed      bool
	errMessage     string
	editingID      string // empty for add
}

func newRecurringAddFormModel() recurringFormModel {
	return recurringFormModel{
		step:          recurringFieldType,
		typeChoices:   []string{model.TransactionTypeExpense, model.TransactionTypeIncome, model.TransactionTypeTransfer},
		currencyInput: "CAD",
		startInput:    time.Now().Format("2006-01-02"),
		dayInput:      "1",
	}
}

func newRecurringEditFormModel(rule model.RecurringRule) recurringFormModel {
	typeIdx := 0
	choices := []string{model.TransactionTypeExpense, model.TransactionTypeIncome, model.TransactionTypeTransfer}
	for i, c := range choices {
		if c == rule.Type {
			typeIdx = i
			break
		}
	}
	return recurringFormModel{
		step:           recurringFieldType,
		typeChoices:    choices,
		typeCursor:     typeIdx,
		amountInput:    formatRuleAmountInput(rule.AmountMinor),
		currencyInput:  rule.Currency,
		categoryInput:  rule.Category,
		accountInput:   rule.Account,
		toAccountInput: rule.ToAccount,
		dayInput:       strconv.Itoa(rule.DayOfMonth),
		startInput:     rule.StartDate,
		endInput:       rule.EndDate,
		noteInput:      rule.Note,
		editingID:      rule.ID,
	}
}

func formatRuleAmountInput(amountMinor int64) string {
	whole := amountMinor / 100
	fraction := amountMinor % 100
	return fmt.Sprintf("%d.%02d", whole, fraction)
}

func (m recurringFormModel) Init() tea.Cmd { return nil }

func (m recurringFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
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
			case "enter":
				m.errMessage = ""
				m.step = recurringFieldAccount
			case "esc":
				m.step = recurringFieldCurrency
			case "backspace":
				if len(m.categoryInput) > 0 {
					m.categoryInput = m.categoryInput[:len(m.categoryInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.categoryInput += msg.String()
				}
			}
		case recurringFieldAccount:
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.accountInput) == "" {
					m.errMessage = "account is required"
					break
				}
				m.errMessage = ""
				if m.typeChoices[m.typeCursor] == model.TransactionTypeTransfer {
					m.step = recurringFieldToAccount
				} else {
					m.step = recurringFieldDayOfMonth
				}
			case "esc":
				m.step = recurringFieldCategory
			case "backspace":
				if len(m.accountInput) > 0 {
					m.accountInput = m.accountInput[:len(m.accountInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.accountInput += msg.String()
				}
			}
		case recurringFieldToAccount:
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.toAccountInput) == "" {
					m.errMessage = "destination account is required"
					break
				}
				m.errMessage = ""
				m.step = recurringFieldDayOfMonth
			case "esc":
				m.step = recurringFieldAccount
			case "backspace":
				if len(m.toAccountInput) > 0 {
					m.toAccountInput = m.toAccountInput[:len(m.toAccountInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.toAccountInput += msg.String()
				}
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
		s += mutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case recurringFieldCurrency:
		s += renderField("Type: ", m.typeChoices[m.typeCursor]) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n\n"
		s += renderActiveField("Currency: ", m.currencyInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case recurringFieldCategory:
		s += renderField("Type: ", m.typeChoices[m.typeCursor]) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Currency: ", m.currencyInput) + "\n\n"
		s += renderActiveField("Category: ", m.categoryInput) + "\n"
		s += mutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case recurringFieldAccount:
		s += renderField("Type: ", m.typeChoices[m.typeCursor]) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Currency: ", m.currencyInput) + "\n"
		s += renderField("Category: ", m.categoryInput) + "\n\n"
		label := "Account: "
		if m.typeChoices[m.typeCursor] == model.TransactionTypeTransfer {
			label = "From account: "
		}
		s += renderActiveField(label, m.accountInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case recurringFieldToAccount:
		s += renderField("Type: ", m.typeChoices[m.typeCursor]) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("From account: ", m.accountInput) + "\n\n"
		s += renderActiveField("To account: ", m.toAccountInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case recurringFieldDayOfMonth:
		s += renderField("Type: ", m.typeChoices[m.typeCursor]) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Account: ", m.accountInput) + "\n\n"
		s += renderActiveField("Day of month (1-28): ", m.dayInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case recurringFieldStartDate:
		s += renderField("Day of month: ", m.dayInput) + "\n\n"
		s += renderActiveField("Start date (YYYY-MM-DD): ", m.startInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case recurringFieldEndDate:
		s += renderField("Start date: ", m.startInput) + "\n\n"
		s += renderActiveField("End date (YYYY-MM-DD, blank for none): ", m.endInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case recurringFieldNote:
		s += renderActiveField("Note (optional): ", m.noteInput) + "\n"
		s += mutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case recurringFieldConfirm:
		s += renderField("Type: ", m.typeChoices[m.typeCursor]) + "\n"
		s += renderField("Amount: ", m.amountInput) + " " + m.currencyInput + "\n"
		s += renderField("Category: ", m.categoryInput) + "\n"
		if m.typeChoices[m.typeCursor] == model.TransactionTypeTransfer {
			s += renderField("From -> To: ", m.accountInput+" -> "+m.toAccountInput) + "\n"
		} else {
			s += renderField("Account: ", m.accountInput) + "\n"
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

func runRecurringAdd() (bool, error) {
	p := tea.NewProgram(newRecurringAddFormModel())
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
		Category:    result.categoryInput,
		Account:     result.accountInput,
		ToAccount:   result.toAccountInput,
		Note:        result.noteInput,
		DayOfMonth:  day,
		StartDate:   result.startInput,
		EndDate:     result.endInput,
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

	p := tea.NewProgram(newRecurringEditFormModel(rule))
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
	category := result.categoryInput
	account := result.accountInput
	toAccount := result.toAccountInput
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
