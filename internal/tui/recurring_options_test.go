package tui

import (
	"testing"

	"github.com/amiraminb/coinwarrior/internal/service"
	tea "github.com/charmbracelet/bubbletea"
)

func recurringString(value string) *string { return &value }

func TestRecurringPartialFlagsPromptOnlyMissingFields(t *testing.T) {
	m, err := newRecurringAddModelWithOptions([]string{"Subscriptions", "Utilities"}, []string{"Checking", "Savings"}, service.RecurringRuleEdits{
		Type: recurringString("expense"), AmountInput: recurringString("20"), Account: recurringString("checking"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.step != recurringFieldCategory {
		t.Fatalf("first missing field = %v, want category", m.step)
	}
	enter, esc := tea.KeyMsg{Type: tea.KeyEnter}, tea.KeyMsg{Type: tea.KeyEsc}
	next, _ := m.Update(enter)
	m = next.(recurringFormModel)
	if m.step != recurringFieldDayOfMonth {
		t.Fatalf("next missing field = %v, want day", m.step)
	}
	next, _ = m.Update(esc)
	m = next.(recurringFormModel)
	if m.step != recurringFieldCategory {
		t.Fatalf("Esc should skip supplied account, got %v", m.step)
	}
	next, _ = m.Update(enter)
	m = next.(recurringFormModel)
	m.dayInput = "15"
	next, _ = m.Update(enter)
	m = next.(recurringFormModel)
	if m.step != recurringFieldConfirm || m.confirmed {
		t.Fatal("partially interactive entry should confirm after the last missing field")
	}
	next, _ = m.Update(esc)
	m = next.(recurringFormModel)
	if m.step != recurringFieldDayOfMonth || m.amountInput != "20" || m.selectedAccount() != "Checking" {
		t.Fatalf("back navigation lost supplied values: %+v", m)
	}
}

func TestRecurringTransferKeepsSuppliedDestination(t *testing.T) {
	day := 15
	m, err := newRecurringAddModelWithOptions([]string{"Subscriptions"}, []string{"Checking", "Savings"}, service.RecurringRuleEdits{
		Type: recurringString("transfer"), AmountInput: recurringString("20"),
		ToAccount: recurringString("savings"), DayOfMonth: &day,
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.step != recurringFieldAccount {
		t.Fatalf("transfer should prompt for source, not category: %v", m.step)
	}
	m.accountCursor = 1
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(recurringFormModel)
	if m.step != recurringFieldAccount || m.errMessage == "" || m.selectedToAccount() != "Savings" {
		t.Fatal("choosing the destination as source must not silently change the supplied destination")
	}
	m.accountCursor = 0
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(recurringFormModel)
	if m.step != recurringFieldConfirm || m.selectedToAccount() != "Savings" {
		t.Fatal("a valid source should advance to confirmation, preserving the destination")
	}
}

func TestRecurringBareFormStillPromptsAndTransferSkipsCategory(t *testing.T) {
	m, err := newRecurringAddModelWithOptions([]string{"Subscriptions"}, []string{"Checking", "Savings"}, service.RecurringRuleEdits{})
	if err != nil || m.step != recurringFieldType || len(m.preset) != 0 {
		t.Fatalf("bare add should open the full form: %+v, %v", m, err)
	}
	m.typeCursor = 2 // transfer
	m.step = recurringFieldCurrency
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(recurringFormModel)
	if m.step != recurringFieldAccount {
		t.Fatalf("transfer should skip category, got %v", m.step)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(recurringFormModel).step != recurringFieldCurrency {
		t.Fatal("back navigation should also skip the transfer's category")
	}
}
