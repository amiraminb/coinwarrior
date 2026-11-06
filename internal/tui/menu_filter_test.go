package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/amiraminb/coinwarrior/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectionFiltersAndNavigatesOnlyMatchingRows(t *testing.T) {
	m := newSelectionModel("", "", []selectionItem[int]{
		{label: "Housing", value: 10},
		{label: "Groceries", value: 20},
		{label: "Utilities", value: 30},
		{label: "Grocery delivery", value: 40},
	})
	for _, key := range []string{"down", "down", "down", "G", "r", "O"} {
		next, cmd := m.Update(keyPress(key))
		if quits(cmd) {
			t.Fatal("typing a filter must not quit")
		}
		m = next.(selectionModel[int])
	}
	view := m.View()
	if !strings.Contains(view, "> Groceries") || !strings.Contains(view, "Grocery delivery") ||
		strings.Contains(view, "Housing") || strings.Contains(view, "Utilities") {
		t.Fatalf("expected only case-insensitive substring matches, got:\n%s", view)
	}
	for _, key := range []string{"up", "down", "down", "up", "down"} {
		next, _ := m.Update(keyPress(key))
		m = next.(selectionModel[int])
	}
	next, cmd := m.Update(keyPress("enter"))
	if got := next.(selectionModel[int]); !got.chose || got.selected != 40 || !quits(cmd) {
		t.Fatalf("Enter must select the highlighted match's original value: %+v", got)
	}
}

func TestSelectionFilterAcceptsLettersSpacesPasteAndUnicode(t *testing.T) {
	m := newSelectionModel("", "", []selectionItem[int]{
		{label: "Other", value: 1},
		{label: "Qjk café", value: 2},
	})
	keys := []tea.KeyMsg{
		keyRunes("q"), keyRunes("j"), keyRunes("k"), {Type: tea.KeySpace},
		{Type: tea.KeyRunes, Runes: []rune("CAFÉx"), Paste: true},
		{Type: tea.KeyBackspace}, {Type: tea.KeyCtrlH}, keyRunes("é"),
	}
	for _, key := range keys {
		next, cmd := m.Update(key)
		if quits(cmd) {
			t.Fatal("filter text must not trigger shortcuts")
		}
		m = next.(selectionModel[int])
	}
	if !strings.Contains(m.View(), "> Qjk café") || strings.Contains(m.View(), "Other") {
		t.Fatalf("expected the accented name after editing pasted text:\n%s", m.View())
	}
	next, _ := m.Update(keyPress("enter"))
	if next.(selectionModel[int]).selected != 2 {
		t.Fatal("filter must preserve the selected value")
	}

	for _, clear := range []tea.KeyType{tea.KeyEsc, tea.KeyCtrlU} {
		next, cmd := m.Update(tea.KeyMsg{Type: clear})
		if quits(cmd) || !strings.Contains(next.View(), "Other") {
			t.Fatal("clearing the filter must restore the full list without quitting")
		}
		if _, cmd = next.Update(keyPress("esc")); !quits(cmd) {
			t.Fatal("Esc with an empty filter must cancel")
		}
	}
}

func TestAllListMenusBlockSelectionWhenNothingMatches(t *testing.T) {
	menus := map[string]tea.Model{
		"shared":      newSelectionModel("", "", []selectionItem[string]{{label: "Groceries", value: "groceries"}}),
		"empty":       newSelectionModel[string]("", "", nil),
		"transaction": newTransactionListModel("", []model.Transaction{expenseTx("Groceries")}),
		"account":     newAccountUpdateModel([]model.Account{{Name: "Checking"}}),
	}
	for name, step := range map[string]addStep{
		"add type": stepType, "add category": stepCategorySelect,
		"add account": stepAccountSelect, "add transfer destination": stepTransferToAccountSelect,
	} {
		m := newAddModel([]string{"Groceries"}, []string{"Checking"})
		m.step = step
		menus[name] = m
	}
	for name, step := range map[string]recurringField{
		"recurring type": recurringFieldType, "recurring category": recurringFieldCategory,
		"recurring account": recurringFieldAccount, "recurring destination": recurringFieldToAccount,
	} {
		m := newRecurringAddFormModel([]string{"Groceries"}, []string{"Checking"})
		m.step = step
		menus[name] = m
	}
	edit := newEditModel(expenseTx("Groceries"), []string{"Groceries"})
	edit.step = editStepCategorySelect
	menus["edit category"] = edit
	budget := newBudgetSetModel([]string{"Groceries"})
	budget.step = budgetSetStepCategory
	menus["budget category"] = budget

	for name, m := range menus {
		t.Run(name, func(t *testing.T) {
			m, _ = m.Update(keyRunes("no-such-option"))
			view := m.View()
			if !strings.Contains(view, "No matches.") {
				t.Fatalf("expected an empty-result message:\n%s", view)
			}
			for _, key := range []string{"down", "up", "enter"} {
				next, cmd := m.Update(keyPress(key))
				if cmd != nil || next.View() != view {
					t.Fatalf("%s must not select a hidden item or leave an empty list", key)
				}
				m = next
			}
			if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC}); !quits(cmd) {
				t.Fatal("Ctrl+C must quit with an active filter")
			}
		})
	}
}

func TestAddFiltersResetBetweenCategoryAndAccount(t *testing.T) {
	m, err := newAddModelWithOptions([]string{"Housing", "Groceries"}, []string{"Checking", "Joint Savings"}, AddTransactionOptions{
		Type: addOption("expense"), Amount: addOption("20"), Date: addOption("2026-05-21"),
	})
	if err != nil {
		t.Fatal(err)
	}
	m = advanceAdd(m, "rOC", "enter")
	if m.step != stepAccountSelect || m.categoryInput != "Groceries" || !strings.Contains(m.View(), "Checking") {
		t.Fatalf("category selection or filter reset failed: %+v", m)
	}
	m = advanceAdd(m, "j", "o", "i", "n", "t", "enter")
	if m.step != stepNote || m.accountInput != "Joint Savings" || m.createAccount {
		t.Fatalf("filtered account must resolve to the saved name: %+v", m)
	}
	m = advanceAdd(m, "esc")
	if !strings.Contains(m.View(), "Checking") || !strings.Contains(m.View(), "> Joint Savings") {
		t.Fatalf("going back must show all accounts and retain the selection:\n%s", m.View())
	}
}

func TestFilteredSpecialRowsKeepTheirMeaning(t *testing.T) {
	for _, tc := range []struct {
		name     string
		current  string
		query    string
		category string
		step     editStep
	}{
		{"saved category", "Housing", "gro", "Groceries", editStepAccount},
		{"literal label", "Housing", "no category", noCategoryLabel, editStepAccount},
		{"clear category", "", "no category", "", editStepAccount},
		{"new category", "Housing", "new", "Housing", editStepCategoryInput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newEditModel(expenseTx(tc.current), []string{"Housing", "Groceries", noCategoryLabel})
			m.step = editStepCategorySelect
			m = advance(m, tc.query, "enter")
			if m.step != tc.step || m.categoryInput != tc.category {
				t.Fatalf("filtered row resolved incorrectly: step=%d category=%q", m.step, m.categoryInput)
			}
		})
	}
	add := newAddModel([]string{"Housing"}, []string{"Checking"})
	add.step = stepCategorySelect
	add = advanceAdd(add, "new", "enter")
	if add.step != stepCategoryInput {
		t.Fatal("filtered new-category option must open category entry")
	}
	add.step = stepAccountSelect
	add = advanceAdd(add, "new", "enter")
	if add.step != stepAccountInput {
		t.Fatal("filtered new-account option must open account entry")
	}

	budget := newBudgetSetModel([]string{"Housing", "Groceries"})
	budget.step = budgetSetStepCategory
	budget = advanceBudget(budget, "gro", "enter")
	if budget.step != budgetSetStepCurrency || budget.categoryInput != "Groceries" {
		t.Fatal("filtered budget category must account for the All spending row")
	}
	budget = advanceBudget(budget, "esc", "all", "enter")
	if budget.categoryInput != "" {
		t.Fatal("filtered All spending row must clear the category")
	}
}

func TestFilteredAccountAndTransactionUseOriginalRecords(t *testing.T) {
	accounts := []model.Account{{Name: "Checking", Currency: "CAD"}, {Name: "Savings", Currency: "USD", BalanceMinor: 12345}}
	account, _ := newAccountUpdateModel(accounts).Update(keyRunes("sav"))
	account, _ = account.Update(keyPress("enter"))
	if got := account.(accountUpdateModel); got.step != accountUpdateStepAmount || got.selectedAccount != accounts[1] {
		t.Fatalf("filtered account selection lost record identity: %+v", got)
	}

	first, second := expenseTx("Housing"), expenseTx("Groceries")
	second.ID, second.Note = "txn_2", "Weekly shop"
	transaction, _ := newTransactionListModel("", []model.Transaction{first, second}).Update(keyRunes("weekly"))
	transaction, cmd := transaction.Update(keyPress("enter"))
	if got := transaction.(transactionListModel); !reflect.DeepEqual(got.selected, second) || !quits(cmd) {
		t.Fatalf("filtering a transaction by its note must select the original record: %+v", got)
	}
}

func TestRecurringFiltersPreserveTransferAccounts(t *testing.T) {
	var m tea.Model = newRecurringAddFormModel([]string{"Housing", "Groceries"}, []string{"Checking", "Savings", "Joint"})
	for _, key := range []string{"tra", "enter", "2", "0", "enter", "enter", "joint", "enter"} {
		m, _ = m.Update(keyPress(key))
	}
	if got := m.(recurringFormModel); got.step != recurringFieldToAccount || got.selectedAccount() != "Joint" {
		t.Fatalf("expected the filtered source account and destination picker: %+v", got)
	}
	m, _ = m.Update(keyRunes("joint"))
	m, _ = m.Update(keyPress("enter"))
	if got := m.(recurringFormModel); got.step != recurringFieldToAccount || got.errMessage == "" {
		t.Fatal("filtering must not allow a transfer to the source account")
	}
	for _, key := range []string{"esc", "sav", "enter"} {
		m, _ = m.Update(keyPress(key))
	}
	if got := m.(recurringFormModel); got.step != recurringFieldDayOfMonth ||
		got.selectedAccount() != "Joint" || got.selectedToAccount() != "Savings" || got.typeChoices[got.typeCursor] != "transfer" {
		t.Fatalf("filtering must preserve independently selected transfer values: %+v", got)
	}

	category := newRecurringAddFormModel([]string{"Housing", "Groceries"}, []string{"Checking", "Savings"})
	category.step = recurringFieldCategory
	m, _ = category.Update(keyRunes("gro"))
	m, _ = m.Update(keyPress("enter"))
	m, _ = m.Update(keyRunes("sav"))
	m, _ = m.Update(keyPress("enter"))
	if got := m.(recurringFormModel); got.step != recurringFieldDayOfMonth || got.selectedCategory() != "Groceries" || got.selectedAccount() != "Savings" {
		t.Fatalf("filter must reset between recurring category and account: %+v", got)
	}
}
