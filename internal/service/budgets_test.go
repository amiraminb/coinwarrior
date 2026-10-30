package service

import (
	"slices"
	"testing"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
)

type budgetRepo struct {
	categoriesRepo
	budgets []model.Budget
}

func (r *budgetRepo) LoadBudgets() ([]model.Budget, error) {
	return slices.Clone(r.budgets), nil
}

func (r *budgetRepo) SaveBudgets(budgets []model.Budget) error {
	r.budgets = slices.Clone(budgets)
	return nil
}

func TestCategoryBudgetUpdatesOnlyItsMonthCurrencyAndCategory(t *testing.T) {
	r := &budgetRepo{categoriesRepo: categoriesRepo{categories: []string{"Groceries", "Dining"}}}
	s := New(r)
	for _, input := range []struct{ month, currency, category, amount string }{
		{"2026-11", "CAD", "", "1000"},
		{"2026-11", "CAD", "Groceries", "200"},
		{"2026-11", "CAD", "Dining", "100"},
		{"2026-11", "USD", "Groceries", "50"},
		{"2026-12", "CAD", "Groceries", "300"},
		{"2026-11", "cad", " groceries ", "250"},
	} {
		if _, err := s.SetMonthlyCategoryBudget(input.month, input.currency, input.category, input.amount); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.budgets) != 5 {
		t.Fatalf("got %d budgets, want 5 distinct limits", len(r.budgets))
	}
	for i, want := range []int64{100000, 25000, 10000, 5000, 30000} {
		if r.budgets[i].AmountMinor != want {
			t.Errorf("budget %d = %d, want %d", i, r.budgets[i].AmountMinor, want)
		}
	}
	if r.budgets[1].Category != "Groceries" {
		t.Error("category should use the saved spelling")
	}
	if _, err := s.SetMonthlyCategoryBudget("2026-11", "CAD", "Unknown", "100"); err == nil {
		t.Fatal("unknown category should be rejected")
	}
	if _, err := s.SetMonthlyBudget("2026-11", "CAD", "1200"); err != nil {
		t.Fatal(err)
	}
	if len(r.budgets) != 5 || r.budgets[0].AmountMinor != 120000 || r.budgets[1].AmountMinor != 25000 {
		t.Error("legacy overall setter must update only the overall budget")
	}
}

func TestCategoryBudgetSummariesCountOnlyMatchingExpenses(t *testing.T) {
	budgets := []model.Budget{
		{Month: "2026-11", Currency: "CAD", AmountMinor: 100000},
		{Month: "2026-11", Currency: "CAD", Category: "Groceries", AmountMinor: 50000},
		{Month: "2026-11", Currency: "USD", Category: "Groceries", AmountMinor: 10000},
		{Month: "2026-11", Currency: "CAD", Category: "Dining", AmountMinor: 1000},
	}
	txs := []model.Transaction{
		{Type: "expense", Date: "2026-11-01", Currency: "CAD", Category: "groceries", AmountMinor: 2000},
		{Type: "expense", Date: "2026-11-30", Currency: "CAD", Category: "Groceries", AmountMinor: 1000},
		{Type: "expense", Date: "2026-11-10", Currency: "CAD", Category: "Dining", AmountMinor: 4000},
		{Type: "expense", Date: "2026-11-10", Currency: "USD", Category: "Groceries", AmountMinor: 5000},
		{Type: "income", Date: "2026-11-10", Currency: "CAD", Category: "Groceries", AmountMinor: 100},
		{Type: "transfer", Date: "2026-11-10", Currency: "CAD", Category: "Groceries", AmountMinor: 50000},
		{Type: "expense", Date: "2026-10-31", Currency: "CAD", Category: "Groceries", AmountMinor: 99999},
		{Type: "expense", Date: "2026-12-01", Currency: "CAD", Category: "Groceries", AmountMinor: 99999},
	}
	month := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	summaries, err := summarizeBudgetsForMonth(budgets, txs, month, month)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 4 {
		t.Fatalf("got %d summaries, want 4", len(summaries))
	}
	want := map[string]int64{"CAD/": 7000, "CAD/Groceries": 3000, "CAD/Dining": 4000, "USD/Groceries": 5000}
	for _, summary := range summaries {
		key := summary.Budget.Currency + "/" + summary.Budget.Category
		if summary.SpentMinor != want[key] || summary.LeftMinor != summary.Budget.AmountMinor-want[key] {
			t.Errorf("incorrect summary for %s: %+v", key, summary)
		}
	}
}

func TestCategoryRolloverNeverUsesAnotherCategoryOrOverallBudget(t *testing.T) {
	for _, carry := range []bool{true, false} {
		r := &budgetRepo{
			categoriesRepo: categoriesRepo{
				categories: []string{"Groceries", "Dining"},
				fakeRepo: fakeRepo{transactions: []model.Transaction{
					{Type: "expense", Date: "2026-01-10", Currency: "CAD", Category: "Groceries", AmountMinor: 2500},
					{Type: "expense", Date: "2026-01-10", Currency: "CAD", Category: "Dining", AmountMinor: 9000},
				}},
			},
			budgets: []model.Budget{
				{Month: "2026-01", Currency: "CAD", AmountMinor: 50000},
				{Month: "2026-01", Currency: "CAD", Category: "Groceries", AmountMinor: 10000},
				{Month: "2026-01", Currency: "CAD", Category: "Dining", AmountMinor: 10000},
				{Month: "2026-02", Currency: "CAD", AmountMinor: 50000},
			},
		}
		s := New(r)
		now := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)
		candidate, err := s.GetCategoryBudgetCarryoverCandidate("2026-02", "CAD", "groceries", now)
		if err != nil || candidate == nil || candidate.LeftMinor != 7500 {
			t.Fatalf("candidate = %+v, error = %v; want 7500", candidate, err)
		}
		if _, err := s.setMonthlyBudgetWithNow("2026-02", "CAD", "Groceries", "150", &carry, now); err != nil {
			t.Fatal(err)
		}
		if r.budgets[0].RolloverStatus != "" || r.budgets[2].RolloverStatus != "" || r.budgets[3].RolloverMinor != 0 {
			t.Fatal("category rollover modified another budget")
		}
		want := int64(0)
		if carry {
			want = 7500
		}
		if r.budgets[4].RolloverMinor != want || r.budgets[4].Category != "Groceries" {
			t.Errorf("carry=%v: unexpected destination: %+v", carry, r.budgets[4])
		}
	}
}

func TestApplyCategoryRolloverCreatesMatchingDestination(t *testing.T) {
	r := &budgetRepo{budgets: []model.Budget{
		{Month: "2026-01", Currency: "CAD", Category: "Travel", AmountMinor: 40000},
		{Month: "2026-02", Currency: "CAD", AmountMinor: 350000},
	}}
	s := New(r)
	now := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)
	_, dest, err := s.ApplyMonthlyCategoryBudgetRollover("2026-01", "CAD", "travel", true, now)
	if err != nil {
		t.Fatal(err)
	}
	if dest == nil || dest.Category != "Travel" || dest.Month != "2026-02" || dest.RolloverMinor != 40000 {
		t.Fatalf("unexpected destination: %+v", dest)
	}
	if r.budgets[1].RolloverMinor != 0 {
		t.Error("travel funds leaked into the overall budget")
	}
	if _, _, err := s.ApplyMonthlyCategoryBudgetRollover("2026-01", "CAD", "Travel", true, now); err == nil {
		t.Error("a second rollover must be rejected")
	}
}
