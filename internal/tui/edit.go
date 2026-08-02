package tui

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
	"github.com/amiraminb/coinwarrior/internal/service"
	tea "github.com/charmbracelet/bubbletea"
)

const noCategoryChoice = "(no category)"

type editStep int

const (
	editStepDate editStep = iota
	editStepAmount
	editStepCategorySelect
	editStepCategoryInput
	editStepCategoryConfirm
	editStepAccount
	editStepToAccount
	editStepNote
	editStepConfirm
	editStepDone
)

type editModel struct {
	step editStep

	selected       model.Transaction
	dateInput      string
	amountInput    string
	categoryInput  string
	accountInput   string
	toAccountInput string
	noteInput      string

	categories      []string
	categoryCursor  int
	categoryDraft   string
	pendingCategory string
	categoryConfirm int

	confirmCursor int
	confirmed     bool
	errMessage    string
}

func newEditModel(selected model.Transaction, categories []string) editModel {
	categories, cursor := editCategoryChoices(categories, selected.Category)

	return editModel{
		step:           editStepDate,
		selected:       selected,
		dateInput:      selected.Date,
		amountInput:    formatEditAmountInput(selected.AmountMinor),
		categoryInput:  selected.Category,
		accountInput:   selected.Account,
		toAccountInput: selected.ToAccount,
		noteInput:      selected.Note,
		categories:     categories,
		categoryCursor: cursor,
	}
}

// Transfers skip the category steps: they always carry model.TransferCategory
// and are excluded from every category report, so the choice would be inert.
func (m editModel) stepAfterAmount() editStep {
	if m.selected.Type == model.TransactionTypeTransfer {
		return editStepAccount
	}
	return editStepCategorySelect
}

func (m editModel) Init() tea.Cmd {
	return nil
}

// isSelectionStep reports whether the step is a menu (list or Yes/No) rather
// than free-text entry. On menu steps a bare "q" quits; on text steps it must
// be typed into the field, so the global handler lets it fall through.
func (s editStep) isSelectionStep() bool {
	switch s {
	case editStepCategorySelect, editStepCategoryConfirm, editStepConfirm:
		return true
	default:
		return false
	}
}

func (m editModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				return m, tea.Quit
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
				amountMinor, err := money.Parse(amount)
				if err != nil {
					m.errMessage = err.Error()
					break
				}
				if amountMinor <= 0 {
					m.errMessage = "amount must be greater than zero"
					break
				}
				m.errMessage = ""
				m.step = m.stepAfterAmount()
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
		case editStepCategorySelect:
			switch msg.String() {
			case "up", "k":
				if m.categoryCursor > 0 {
					m.categoryCursor--
				}
			case "down", "j":
				if m.categoryCursor < len(m.categories) {
					m.categoryCursor++
				}
			case "enter":
				m.errMessage = ""
				if m.categoryCursor < len(m.categories) {
					picked := m.categories[m.categoryCursor]
					if picked == noCategoryChoice {
						picked = ""
					}
					m.categoryInput = picked
					m.step = editStepAccount
					break
				}
				m.categoryDraft = ""
				m.step = editStepCategoryInput
			case "esc":
				m.errMessage = ""
				m.step = editStepAmount
			}
		case editStepCategoryInput:
			switch msg.String() {
			case "enter":
				draft := strings.TrimSpace(m.categoryDraft)
				if draft == "" {
					m.errMessage = "category is required"
					break
				}
				m.errMessage = ""
				if i := slices.IndexFunc(m.categories, func(c string) bool { return strings.EqualFold(c, draft) }); i >= 0 {
					m.categoryInput = m.categories[i]
					m.step = editStepAccount
					break
				}
				m.pendingCategory = draft
				m.categoryConfirm = 0
				m.step = editStepCategoryConfirm
			case "esc":
				m.errMessage = ""
				m.step = editStepCategorySelect
			case "backspace":
				m.errMessage = ""
				if len(m.categoryDraft) > 0 {
					m.categoryDraft = m.categoryDraft[:len(m.categoryDraft)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.categoryDraft += msg.String()
					m.errMessage = ""
				}
			}
		case editStepCategoryConfirm:
			switch msg.String() {
			case "left", "h", "up", "k":
				m.categoryConfirm = 0
			case "right", "l", "down", "j":
				m.categoryConfirm = 1
			case "enter":
				if m.categoryConfirm == 0 {
					m.categoryInput = m.pendingCategory
					m.step = editStepAccount
					break
				}
				m.step = editStepCategoryInput
			case "esc":
				m.step = editStepCategoryInput
			}
		case editStepAccount:
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.accountInput) == "" {
					m.errMessage = "account is required"
					break
				}
				m.errMessage = ""
				if m.selected.Type == model.TransactionTypeTransfer {
					m.step = editStepToAccount
				} else {
					m.step = editStepNote
				}
			case "esc":
				m.errMessage = ""
				if m.selected.Type == model.TransactionTypeTransfer {
					m.step = editStepAmount
				} else {
					m.step = editStepCategorySelect
				}
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
				if m.selected.Type == model.TransactionTypeTransfer {
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
	case editStepDate:
		s += renderField("Editing: ", m.selected.ID) + "\n"
		s += renderField("Type: ", m.selected.Type) + "\n\n"
		s += renderActiveField("Date: ", m.dateInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to cancel, ctrl+c to quit)") + "\n"
	case editStepAmount:
		s += renderField("Editing: ", m.selected.ID) + "\n"
		s += renderField("Date: ", m.dateInput) + "\n\n"
		s += renderActiveField("Amount: ", m.amountInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case editStepCategorySelect:
		s += renderField("Editing: ", m.selected.ID) + "\n"
		s += renderField("Date: ", m.dateInput) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Current category: ", editCategoryLabel(m.selected.Category)) + "\n\n"
		s += "Select category:\n\n"
		for i, c := range m.categories {
			line := "  " + c
			if i == m.categoryCursor {
				line = focusStyle.Render("> " + c)
			}
			s += line + "\n"
		}
		newOptionLine := "  [New category]"
		if m.categoryCursor == len(m.categories) {
			newOptionLine = focusStyle.Render("> [New category]")
		}
		s += newOptionLine + "\n"
		s += "\n" + mutedStyle.Render("(use ↑/↓ and enter, esc to go back, q to quit)") + "\n"
	case editStepCategoryInput:
		s += renderField("Editing: ", m.selected.ID) + "\n"
		s += renderField("Date: ", m.dateInput) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n\n"
		s += renderActiveField("Enter category: ", m.categoryDraft) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case editStepCategoryConfirm:
		s += renderField("Editing: ", m.selected.ID) + "\n"
		s += renderField("Date: ", m.dateInput) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n\n"
		s += warnStyle.Render("Category '"+m.pendingCategory+"' is new. Create it?") + "\n\n"
		s += renderYesNo(m.categoryConfirm == 0) + "\n"
		s += mutedStyle.Render("(use ←/→ or ↑/↓ and enter)") + "\n"
	case editStepAccount:
		s += renderField("Editing: ", m.selected.ID) + "\n"
		s += renderField("Date: ", m.dateInput) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Category: ", m.categoryInput) + "\n\n"
		s += renderActiveField("Account: ", m.accountInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case editStepToAccount:
		s += renderField("Editing: ", m.selected.ID) + "\n"
		s += renderField("Date: ", m.dateInput) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Category: ", m.categoryInput) + "\n"
		s += renderField("From account: ", m.accountInput) + "\n\n"
		s += renderActiveField("To account: ", m.toAccountInput) + "\n"
		s += renderError(m.errMessage)
		s += mutedStyle.Render("(enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case editStepNote:
		s += renderField("Editing: ", m.selected.ID) + "\n"
		s += renderField("Date: ", m.dateInput) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Category: ", m.categoryInput) + "\n"
		s += renderField("Account: ", m.accountInput) + "\n"
		if m.selected.Type == model.TransactionTypeTransfer {
			s += renderField("To account: ", m.toAccountInput) + "\n"
		}
		s += "\n" + renderActiveField("Note: ", m.noteInput) + "\n"
		s += mutedStyle.Render("(enter to continue, esc to go back, ctrl+c to quit)") + "\n"
	case editStepConfirm:
		s += renderField("Editing: ", m.selected.ID) + "\n"
		s += renderField("Date: ", m.dateInput) + "\n"
		s += renderField("Amount: ", m.amountInput) + "\n"
		s += renderField("Category: ", m.categoryInput) + "\n"
		s += renderField("Account: ", m.accountInput) + "\n"
		if m.selected.Type == model.TransactionTypeTransfer {
			s += renderField("To account: ", m.toAccountInput) + "\n"
		}
		s += renderField("Note: ", m.noteInput) + "\n\n"
		s += warnStyle.Render("Save transaction changes?") + "\n\n"
		s += renderYesNo(m.confirmCursor == 0) + "\n"
		s += mutedStyle.Render("(use ←/→ or ↑/↓, enter to confirm, esc to go back, q to quit)") + "\n"
	case editStepDone:
		s += mutedStyle.Render("Done") + "\n"
	}

	return s
}

func RunEditTransaction() error {
	selected, ok, err := SelectTransaction("Edit Transaction")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("edit cancelled")
		return nil
	}

	categories, err := Svc.LoadCategories()
	if err != nil {
		return err
	}

	p := tea.NewProgram(newEditModel(selected, categories))
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	result := finalModel.(editModel)
	if !result.confirmed || result.selected.ID == "" {
		fmt.Println("edit cancelled")
		return nil
	}

	if result.selected.Type != model.TransactionTypeTransfer {
		if err := Svc.AddCategory(result.categoryInput); err != nil {
			return err
		}
	}

	date := result.dateInput
	amount := result.amountInput
	category := result.categoryInput
	account := result.accountInput
	note := result.noteInput

	edits := service.TransactionEdits{
		Date:     &date,
		Amount:   &amount,
		Category: &category,
		Account:  &account,
		Note:     &note,
	}
	if result.selected.Type == model.TransactionTypeTransfer {
		toAccount := result.toAccountInput
		edits.ToAccount = &toAccount
	}

	tx, err := Svc.EditTransaction(result.selected.ID, edits)
	if err != nil {
		return err
	}

	fmt.Printf("updated transaction: %s\n", tx.ID)
	return nil
}

func SortTransactionsByDateDesc(transactions []model.Transaction) {
	slices.SortFunc(transactions, func(a, b model.Transaction) int {
		if a.Date != b.Date {
			return cmp.Compare(b.Date, a.Date)
		}
		return cmp.Compare(b.CreatedAt, a.CreatedAt)
	})
}

func FormatEditableTransaction(tx model.Transaction) string {
	amount := money.FormatTransaction(tx)

	category := strings.TrimSpace(tx.Category)
	if category == "" {
		category = "(no category)"
	}

	details := tx.Account
	if tx.Type == model.TransactionTypeTransfer {
		details = tx.Account + " -> " + tx.ToAccount
	}

	label := fmt.Sprintf("%s | %s %s | %s | %s | %s", tx.Date, amount, tx.Currency, tx.Type, category, details)
	if strings.TrimSpace(tx.Note) != "" {
		label += " | " + strings.TrimSpace(tx.Note)
	}

	return label
}

// An unlisted stored category is prepended rather than left out, where a single
// enter on the wrong pre-selected entry would silently reassign it.
func editCategoryChoices(categories []string, current string) ([]string, int) {
	choices := slices.Clone(categories)

	if i := slices.IndexFunc(choices, func(c string) bool { return strings.EqualFold(c, current) }); i >= 0 {
		return choices, i
	}
	if strings.TrimSpace(current) == "" {
		return append([]string{noCategoryChoice}, choices...), 0
	}

	return append([]string{current}, choices...), 0
}

func editCategoryLabel(category string) string {
	if strings.TrimSpace(category) == "" {
		return "(no category)"
	}
	return category
}

func formatEditAmountInput(amountMinor int64) string {
	whole := amountMinor / 100
	fraction := amountMinor % 100
	return fmt.Sprintf("%d.%02d", whole, fraction)
}
