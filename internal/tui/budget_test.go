package tui

import (
	"strings"
	"testing"
)

func advanceBudget(m budgetSetModel, keys ...string) budgetSetModel {
	for _, key := range keys {
		next, _ := m.Update(keyPress(key))
		m = next.(budgetSetModel)
	}
	return m
}

func TestBudgetFormSelectsAndDisplaysCategory(t *testing.T) {
	m := newBudgetSetModel([]string{"Groceries", "Dining"})
	m = advanceBudget(m, "enter")
	if m.step != budgetSetStepCategory || !strings.Contains(m.View(), "All spending") || !strings.Contains(m.View(), "Groceries") {
		t.Fatalf("missing category picker: %s", m.View())
	}
	m = advanceBudget(m, "down", "enter", "enter", "5", "0", "0", "enter")
	if m.step != budgetSetStepConfirm || m.categoryInput != "Groceries" || !strings.Contains(m.View(), "Groceries") {
		t.Fatalf("confirmation lost the selected category: %+v", m)
	}
	m = advanceBudget(m, "enter")
	if !m.confirmed {
		t.Error("expected the budget to be confirmed")
	}
}

func TestBudgetFormSupportsOverallAndBackNavigation(t *testing.T) {
	m := newBudgetSetModel([]string{"Groceries"})
	m = advanceBudget(m, "enter", "down", "enter", "esc")
	if m.step != budgetSetStepCategory {
		t.Fatal("Esc from currency must return to category")
	}
	m = advanceBudget(m, "up", "enter")
	if m.categoryInput != "" {
		t.Error("All spending must clear the category")
	}
	m = advanceBudget(m, "esc")
	if _, cmd := m.Update(keyPress("q")); !quits(cmd) {
		t.Error("q must cancel the category picker")
	}
}
