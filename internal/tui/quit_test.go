package tui

import (
	"testing"

	"github.com/amiraminb/coinwarrior/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// quits reports whether a command, when run, yields a tea.QuitMsg.
func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// On text-entry steps, "q" must be typed into the field rather than quitting.
// List menus also accept "q" as filter text. Only Yes/No confirmations retain
// the quit shortcut. Ctrl+C always quits (regression guard for issue #13).

func TestAddModelQuitOnlyOnConfirmationSteps(t *testing.T) {
	for _, step := range []addStep{
		stepCategoryConfirm, stepAccountConfirm,
	} {
		m := newAddModel(nil, nil)
		m.step = step
		if _, cmd := m.Update(keyRunes("q")); !quits(cmd) {
			t.Errorf("q on confirmation step %d should quit", step)
		}
	}

	// All text steps must not quit on "q" (the core of issue #13).
	for _, step := range []addStep{
		stepAmount, stepDate, stepCategoryInput, stepAccountInput, stepNote,
		stepType, stepCategorySelect, stepAccountSelect, stepTransferToAccountSelect,
	} {
		m := newAddModel(nil, nil)
		m.step = step
		if _, cmd := m.Update(keyRunes("q")); quits(cmd) {
			t.Errorf("q on text/filter step %d should not quit", step)
		}
	}

	// On free-text fields that accept letters, "q" is captured as a character.
	capture := []struct {
		step  addStep
		field func(addModel) string
	}{
		{stepCategoryInput, func(m addModel) string { return m.categoryDraft }},
		{stepAccountInput, func(m addModel) string { return m.accountDraft }},
		{stepNote, func(m addModel) string { return m.noteInput }},
	}
	for _, c := range capture {
		m := newAddModel(nil, nil)
		m.step = c.step
		next, _ := m.Update(keyRunes("q"))
		// Currency uppercases input; the others keep it as-is.
		got := c.field(next.(addModel))
		if got != "q" && got != "Q" {
			t.Errorf("q on text step %d was not typed into the field (got %q)", c.step, got)
		}
	}

	m := newAddModel(nil, nil)
	m.step = stepNote
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC}); !quits(cmd) {
		t.Error("ctrl+c on a text step should quit")
	}
}

func TestEditModelQuitOnlyOnConfirm(t *testing.T) {
	tx := model.Transaction{ID: "txn_1", Type: model.TransactionTypeExpense, Date: "2026-03-01"}
	categories := []string{"Groceries", "Dining"}

	for _, step := range []editStep{editStepCategoryConfirm, editStepConfirm} {
		m := newEditModel(tx, categories)
		m.step = step
		if _, cmd := m.Update(keyRunes("q")); !quits(cmd) {
			t.Errorf("q on edit confirmation step %d should quit", step)
		}
	}

	// editStepDate and editStepAmount are numeric/date text steps: "q" is not a
	// valid character there, so it is dropped, but it must still never quit.
	for _, step := range []editStep{editStepDate, editStepAmount, editStepToAccount, editStepCategorySelect} {
		m := newEditModel(tx, categories)
		m.step = step
		if _, cmd := m.Update(keyRunes("q")); quits(cmd) {
			t.Errorf("q on edit text step %d should not quit", step)
		}
	}

	for _, step := range []editStep{editStepCategoryInput, editStepAccount, editStepNote} {
		m := newEditModel(tx, categories)
		m.step = step
		next, cmd := m.Update(keyRunes("q"))
		if quits(cmd) {
			t.Errorf("q on edit text step %d should not quit", step)
		}
		got := next.(editModel)
		field := map[editStep]string{
			editStepCategoryInput: got.categoryDraft,
			editStepAccount:       got.accountInput,
			editStepNote:          got.noteInput,
		}[step]
		if field != "q" {
			t.Errorf("q on edit text step %d was not typed (field=%q)", step, field)
		}
	}
}

func TestAccountAddModelNeverQuitsOnQ(t *testing.T) {
	m := newAccountAddModel()
	m.step = accountStepName
	next, cmd := m.Update(keyRunes("q"))
	if quits(cmd) {
		t.Error("q on account name step should not quit")
	}
	if next.(accountAddModel).nameInput != "q" {
		t.Error("q should be typed into the account name field")
	}

	fresh := newAccountAddModel()
	if _, cmd := fresh.Update(tea.KeyMsg{Type: tea.KeyCtrlC}); !quits(cmd) {
		t.Error("ctrl+c on account add should quit")
	}
}

func TestAccountUpdateModelQuitMatrix(t *testing.T) {
	accounts := []model.Account{{Name: "Checking", Currency: "CAD"}}

	m := newAccountUpdateModel(accounts)
	m.step = accountUpdateStepSelect
	if _, cmd := m.Update(keyRunes("q")); quits(cmd) {
		t.Error("q on account select step should filter, not quit")
	}
	m.step = accountUpdateStepConfirm
	if _, cmd := m.Update(keyRunes("q")); !quits(cmd) {
		t.Error("q on account confirmation should quit")
	}

	m = newAccountUpdateModel(accounts)
	m.step = accountUpdateStepAmount
	next, cmd := m.Update(keyRunes("q"))
	if quits(cmd) {
		t.Error("q on account update amount step should not quit")
	}
	if next.(accountUpdateModel).amountInput != "" {
		t.Error("q (non-numeric) should be ignored by amount field, not quit")
	}
}

func TestRecurringFormModelQuitMatrix(t *testing.T) {
	accounts := []string{"Checking", "Savings"}
	categories := []string{"Rent", "Groceries"}

	m := newRecurringAddFormModel(categories, accounts)
	m.step = recurringFieldConfirm
	if _, cmd := m.Update(keyRunes("q")); !quits(cmd) {
		t.Error("q on recurring confirmation should quit")
	}

	for _, step := range []recurringField{
		recurringFieldAmount, recurringFieldCurrency, recurringFieldDayOfMonth,
		recurringFieldStartDate, recurringFieldEndDate, recurringFieldNote,
		recurringFieldType, recurringFieldCategory, recurringFieldAccount, recurringFieldToAccount,
	} {
		m := newRecurringAddFormModel(categories, accounts)
		m.step = step
		if _, cmd := m.Update(keyRunes("q")); quits(cmd) {
			t.Errorf("q on recurring text step %d should not quit", step)
		}
	}

	// "q" is typed into a free-text field (note), and ctrl+c quits from a menu.
	m = newRecurringAddFormModel(categories, accounts)
	m.step = recurringFieldNote
	next, _ := m.Update(keyRunes("q"))
	if next.(recurringFormModel).noteInput != "q" {
		t.Error("q should be typed into the recurring note field")
	}

	m = newRecurringAddFormModel(categories, accounts)
	m.step = recurringFieldType
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC}); !quits(cmd) {
		t.Error("ctrl+c on a recurring menu step should quit")
	}
}

func TestBudgetSetModelQuitMatrix(t *testing.T) {
	m := newBudgetSetModel(nil)
	m.step = budgetSetStepConfirm
	if _, cmd := m.Update(keyRunes("q")); !quits(cmd) {
		t.Error("q on budget confirm step should quit")
	}

	m = newBudgetSetModel(nil)
	m.step = budgetSetStepCurrency
	next, cmd := m.Update(keyRunes("q"))
	if quits(cmd) {
		t.Error("q on budget currency step should not quit")
	}
	if next.(budgetSetModel).currencyInput != "CADQ" {
		t.Errorf("q should be typed (uppercased) into currency field, got %q", next.(budgetSetModel).currencyInput)
	}
}
