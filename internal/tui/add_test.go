package tui

import (
	"strings"
	"testing"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/amiraminb/coinwarrior/internal/service"
	tea "github.com/charmbracelet/bubbletea"
)

func addOption(value string) *string { return &value }

func completeAddOptions() AddTransactionOptions {
	return AddTransactionOptions{
		Type: addOption("expense"), Amount: addOption("20"), Date: addOption("2026-05-21"),
		Category: addOption("Housing"), Account: addOption("Checking"), Note: addOption(""),
	}
}

func addModelWithOptions(t *testing.T, options AddTransactionOptions) addModel {
	t.Helper()
	m, err := newAddModelWithOptions([]string{"Housing"}, []string{"Checking"}, options)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func advanceAdd(m addModel, keys ...string) addModel {
	for _, key := range keys {
		next, _ := m.Update(keyPress(key))
		m = next.(addModel)
	}
	return m
}

func TestAddOptionsPromptForEachMissingField(t *testing.T) {
	for _, tc := range []struct {
		name string
		omit func(*AddTransactionOptions)
		step addStep
		keys []string
	}{
		{"type", func(o *AddTransactionOptions) { o.Type = nil }, stepType, nil},
		{"amount", func(o *AddTransactionOptions) { o.Amount = nil }, stepAmount, []string{"2", "0"}},
		{"date", func(o *AddTransactionOptions) { o.Date = nil }, stepDate, nil},
		{"category", func(o *AddTransactionOptions) { o.Category = nil }, stepCategorySelect, nil},
		{"account", func(o *AddTransactionOptions) { o.Account = nil }, stepAccountSelect, nil},
		{"note", func(o *AddTransactionOptions) { o.Note = nil }, stepNote, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := completeAddOptions()
			tc.omit(&options)
			m := addModelWithOptions(t, options)
			if m.step != tc.step {
				t.Fatalf("first prompt = %d, want %d", m.step, tc.step)
			}
			m = advanceAdd(m, tc.keys...)
			next, cmd := m.Update(keyPress("enter"))
			if next.(addModel).step != stepDone || !quits(cmd) {
				t.Errorf("completing the only missing field should finish the form, got step %d", next.(addModel).step)
			}
		})
	}
}

func TestAddOptionsNormalizeSuppliedFieldsAndSkipTheForm(t *testing.T) {
	options := completeAddOptions()
	options.Type = addOption(" EXPENSE ")
	options.Category = addOption(" housing ")
	options.Account = addOption(" checking ")
	options.Note = addOption(" rent ")
	m := addModelWithOptions(t, options)
	if m.step != stepDone {
		t.Fatalf("all fields supplied: got step %d, want done", m.step)
	}
	if m.selected != "expense" || m.amountInput != "20" || m.dateInput != "2026-05-21" ||
		m.categoryInput != "Housing" || m.accountInput != "Checking" || m.noteInput != "rent" {
		t.Errorf("supplied values were lost or not normalized: %+v", m)
	}
}

func TestAddOptionsStartAtCategoryAndContinueThroughMissingFields(t *testing.T) {
	options := AddTransactionOptions{
		Type: addOption("expense"), Amount: addOption("20"), Date: addOption("2026-05-21"),
	}
	m := addModelWithOptions(t, options)
	for _, step := range []addStep{stepCategorySelect, stepAccountSelect, stepNote} {
		if m.step != step {
			t.Fatalf("got step %d, want %d", m.step, step)
		}
		m = advanceAdd(m, "enter")
	}
	if m.step != stepDone || m.categoryInput != "Housing" || m.accountInput != "Checking" {
		t.Errorf("missing fields were not collected: %+v", m)
	}
}

func TestAddWithNoOptionsKeepsTheInteractiveFlow(t *testing.T) {
	m := addModelWithOptions(t, AddTransactionOptions{})
	if m.step != stepType {
		t.Fatalf("got step %d, want type", m.step)
	}
	m = advanceAdd(m, "enter", "2", "0", "enter", "enter", "enter", "enter")
	if m.step != stepNote {
		t.Fatalf("got step %d, want note after collecting the required fields", m.step)
	}
	m = advanceAdd(m, "enter")
	if m.step != stepDone {
		t.Errorf("got step %d, want done", m.step)
	}
}

func TestAddBackNavigationSkipsSuppliedFields(t *testing.T) {
	options := completeAddOptions()
	options.Category, options.Note = nil, nil
	m := addModelWithOptions(t, options)
	m = advanceAdd(m, "enter", "esc")
	if m.step != stepCategorySelect {
		t.Errorf("Esc should skip the supplied account and return to category, got step %d", m.step)
	}

	options = completeAddOptions()
	options.Date = nil
	m = advanceAdd(addModelWithOptions(t, options), "esc")
	if m.step != stepDate {
		t.Errorf("Esc should stay at date when all earlier fields were supplied, got step %d", m.step)
	}
}

func TestAddOptionsAllowCreatingMissingCategoriesAndAccounts(t *testing.T) {
	options := completeAddOptions()
	options.Category = nil
	m := addModelWithOptions(t, options)
	m = advanceAdd(m, "down", "enter", "S", "p", "a", "enter")
	if m.step != stepCategoryConfirm {
		t.Fatalf("new category should need confirmation, got step %d", m.step)
	}
	m = advanceAdd(m, "enter")
	if m.step != stepDone || m.categoryInput != "Spa" {
		t.Errorf("new category should complete the form, got %+v", m)
	}

	options = completeAddOptions()
	options.Account = nil
	m = addModelWithOptions(t, options)
	m = advanceAdd(m, "down", "enter", "C", "a", "s", "h", "enter", "enter")
	if m.step != stepDone || m.accountInput != "Cash" || !m.createAccount {
		t.Errorf("new account should complete the form, got %+v", m)
	}
}

func TestAddTypingAnExistingAccountStillPromptsForAMissingNote(t *testing.T) {
	options := completeAddOptions()
	options.Account, options.Note = nil, nil
	m := addModelWithOptions(t, options)
	m = advanceAdd(m, "down", "enter", "C", "h", "e", "c", "k", "i", "n", "g", "enter")
	if m.step != stepNote {
		t.Errorf("typing an existing account should continue to the missing note, got step %d", m.step)
	}
}

func TestAddOptionsRejectInvalidSuppliedFields(t *testing.T) {
	for _, tc := range []struct {
		options AddTransactionOptions
		want    string
	}{
		{AddTransactionOptions{Type: addOption("transfer")}, "--type"},
		{AddTransactionOptions{Type: addOption("")}, "--type"},
		{AddTransactionOptions{Amount: addOption("oops")}, "--amount"},
		{AddTransactionOptions{Amount: addOption("0")}, "--amount"},
		{AddTransactionOptions{Amount: addOption("-20")}, "--amount"},
		{AddTransactionOptions{Amount: addOption("20.001")}, "--amount"},
		{AddTransactionOptions{Date: addOption("2026-02-30")}, "--date"},
		{AddTransactionOptions{Date: addOption("")}, "--date"},
		{AddTransactionOptions{Category: addOption("Unknown")}, "unknown category"},
		{AddTransactionOptions{Category: addOption(" ")}, "--category"},
		{AddTransactionOptions{Account: addOption("Unknown")}, "unknown account"},
		{AddTransactionOptions{Account: addOption("")}, "--account"},
	} {
		_, err := newAddModelWithOptions([]string{"Housing"}, []string{"Checking"}, tc.options)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("got error %v, want %q", err, tc.want)
		}
	}
}

func TestAddCancellationWithPrefilledFieldsDoesNotSave(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := repository.NewFileRepository()
	if err := r.SaveAccounts([]model.Account{{Name: "Checking", Currency: "CAD", BalanceMinor: 10000}}); err != nil {
		t.Fatal(err)
	}
	previous := Svc
	Svc = service.New(r)
	t.Cleanup(func() { Svc = previous })

	options := completeAddOptions()
	options.Note = nil
	m := addModelWithOptions(t, options)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !quits(cmd) {
		t.Fatal("Ctrl+C should quit")
	}
	if err := finishAddTransaction(next.(addModel)); err != nil {
		t.Fatal(err)
	}
	transactions, err := r.LoadTransactions()
	if err != nil {
		t.Fatal(err)
	}
	if len(transactions) != 0 {
		t.Errorf("cancelled entry saved %d transactions", len(transactions))
	}
	accounts, err := r.LoadAccounts()
	if err != nil {
		t.Fatal(err)
	}
	if accounts[0].BalanceMinor != 10000 {
		t.Errorf("cancelled entry changed the account balance to %d", accounts[0].BalanceMinor)
	}
}
