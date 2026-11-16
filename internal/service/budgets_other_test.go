package service

import (
	"testing"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
)

func TestOtherSpendingExcludesRecurringAndBudgetedExpensesByCurrency(t *testing.T) {
	r := &budgetRepo{
		budgets: []model.Budget{
			{Month: "2026-11", Currency: "CAD", AmountMinor: 100000, RolloverMinor: 10000},
			{Month: "2026-11", Currency: "cad", Category: "Groceries", AmountMinor: 40000, RolloverMinor: 5000},
			{Month: "2026-11", Currency: "CAD", Category: "Dining", AmountMinor: 10000, RolloverMinor: -2000},
			{Month: "2026-11", Currency: "USD", AmountMinor: 20000, RolloverMinor: -1000},
			{Month: "2026-11", Currency: "USD", Category: "Groceries", AmountMinor: 5000, RolloverMinor: 2500},
			{Month: "2026-11", Currency: "GBP", Category: "Groceries", AmountMinor: 5000},
			{Month: "2026-10", Currency: "CAD", Category: "Housing", AmountMinor: 99999},
			{Month: "2026-12", Currency: "CAD", Category: "Personal", AmountMinor: 99999},
		},
		categoriesRepo: categoriesRepo{fakeRepo: fakeRepo{
			rules: []model.RecurringRule{
				{ID: "rent", Type: "expense", Currency: "CAD", Category: "Housing", AmountMinor: 25000, DayOfMonth: 1, StartDate: "2026-01-01"},
			},
			transactions: []model.Transaction{
				{Type: "expense", Date: "2026-11-01", Currency: "CAD", Category: " groceries ", AmountMinor: 10000},
				{Type: "expense", Date: "2026-11-02", Currency: "CAD", Category: "dining", AmountMinor: 15000, Source: "recurring"},
				{Type: "expense", Date: "2026-11-03", Currency: "CAD", Category: "Housing", AmountMinor: 20000, Source: "recurring"},
				{Type: "expense", Date: "2026-11-04", Currency: "CAD", Category: "Personal", AmountMinor: 5000},
				{Type: "expense", Date: "2026-11-30", Currency: "CAD", AmountMinor: 1500},
				{Type: "income", Date: "2026-11-05", Currency: "CAD", AmountMinor: 200000},
				{Type: "transfer", Date: "2026-11-05", Currency: "CAD", AmountMinor: 1000},
				{Type: "expense", Date: "2026-10-31", Currency: "CAD", AmountMinor: 99999},
				{Type: "expense", Date: "2026-12-01", Currency: "CAD", AmountMinor: 99999},
				{Type: "expense", Date: "2026-11-01", Currency: "USD", Category: "Groceries", AmountMinor: 2000},
				{Type: "expense", Date: "2026-11-30", Currency: "USD", Category: "Housing", AmountMinor: 1000},
				{Type: "expense", Date: "2026-11-01", Currency: "GBP", Category: "Housing", AmountMinor: 3000, Source: "recurring"},
			},
		}},
	}
	now := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	summaries, err := New(r).GetMonthlyBudgetSummaries("2026-11", now)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct{ budget, roll, spent, left int64 }{
		"CAD": {25000, 7000, 6500, 25500},
		"USD": {15000, -3500, 1000, 10500},
	}
	found := 0
	for _, summary := range summaries {
		if !summary.Other {
			continue
		}
		found++
		expected, ok := want[summary.Budget.Currency]
		if !ok {
			t.Fatalf("unexpected Other spending row without an overall budget: %+v", summary)
		}
		if summary.Budget.AmountMinor != expected.budget || summary.Budget.RolloverMinor != expected.roll ||
			summary.SpentMinor != expected.spent || summary.LeftMinor != expected.left {
			t.Errorf("%s: got %+v, want %+v", summary.Budget.Currency, summary, expected)
		}
		if summary.Recurring || summary.Status != "" || summary.Budget.Category != "" || summary.Budget.RolloverStatus != "" {
			t.Errorf("Other spending must be a separate calculated view: %+v", summary)
		}
		if summary.PeriodStart.Format("2006-01-02") != "2026-11-01" || summary.PeriodEnd.Format("2006-01-02") != "2026-11-30" {
			t.Errorf("unexpected reporting period: %+v", summary)
		}
	}
	if found != len(want) {
		t.Fatalf("got %d Other spending rows, want %d", found, len(want))
	}
}

func TestOtherSpendingReservesRecurringCostsBeforeAndAfterGeneration(t *testing.T) {
	for _, generated := range []bool{false, true} {
		name := "before generation"
		if generated {
			name = "after generation"
		}
		t.Run(name, func(t *testing.T) {
			r := &budgetRepo{
				budgets: []model.Budget{
					{Month: "2026-11", Currency: "CAD", AmountMinor: 400000},
					{Month: "2026-11", Currency: "CAD", Category: "Food Delivery", AmountMinor: 30000},
					{Month: "2026-11", Currency: "CAD", Category: "Groceries", AmountMinor: 80000},
				},
				categoriesRepo: categoriesRepo{fakeRepo: fakeRepo{
					rules: []model.RecurringRule{
						{ID: "rent", Type: "expense", Currency: "CAD", Category: "Rent", AmountMinor: 203700, DayOfMonth: 1, StartDate: "2026-11-01"},
					},
					transactions: []model.Transaction{
						{Type: "expense", Date: "2026-11-01", Currency: "CAD", Category: "Rent", AmountMinor: 10000, Source: "manual"},
						{Type: "expense", Date: "2026-11-02", Currency: "CAD", AmountMinor: 5000},
						{Type: "expense", Date: "2026-11-03", Currency: "CAD", Category: "Groceries", AmountMinor: 20000},
						{Type: "expense", Date: "2026-11-04", Currency: "CAD", Category: "Food Delivery", AmountMinor: 3000},
					},
				}},
			}
			if generated {
				r.rules[0].LastGeneratedMonth = "2026-11"
				r.transactions = append(r.transactions, model.Transaction{
					Type: "expense", Date: "2026-11-01", Currency: "CAD", Category: "Rent", AmountMinor: 203700, Source: "recurring",
				})
			}
			summaries, err := New(r).GetMonthlyBudgetSummaries("2026-11", time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC))
			if err != nil {
				t.Fatal(err)
			}
			for _, summary := range summaries {
				if summary.Other {
					if summary.Budget.AmountMinor != 86300 || summary.SpentMinor != 15000 || summary.LeftMinor != 71300 {
						t.Errorf("want 863.00 budget, 150.00 spent, 713.00 left; got %+v", summary)
					}
					return
				}
			}
			t.Fatal("missing Other spending row")
		})
	}
}

func TestOtherSpendingHandlesUnallocatedAndOverallocatedBudgets(t *testing.T) {
	for _, tc := range []struct {
		name    string
		budgets []model.Budget
		want    int64
	}{
		{
			name: "no category budgets",
			budgets: []model.Budget{
				{Month: "2026-11", Currency: "CAD", AmountMinor: 10000},
			},
			want: 10000,
		},
		{
			name: "fully allocated",
			budgets: []model.Budget{
				{Month: "2026-11", Currency: "CAD", AmountMinor: 10000},
				{Month: "2026-11", Currency: "CAD", Category: "Groceries", AmountMinor: 10000},
			},
			want: 0,
		},
		{
			name: "overallocated",
			budgets: []model.Budget{
				{Month: "2026-11", Currency: "CAD", AmountMinor: 10000},
				{Month: "2026-11", Currency: "CAD", Category: "Groceries", AmountMinor: 12000},
			},
			want: -2000,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &budgetRepo{
				budgets: tc.budgets,
				categoriesRepo: categoriesRepo{fakeRepo: fakeRepo{transactions: []model.Transaction{
					{Type: "expense", Date: "2026-11-01", Currency: "CAD", AmountMinor: 1500},
				}}},
			}
			summaries, err := New(r).GetMonthlyBudgetSummaries("2026-11", time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC))
			if err != nil {
				t.Fatal(err)
			}
			for _, summary := range summaries {
				if summary.Other {
					if summary.Budget.AmountMinor != tc.want || summary.SpentMinor != 1500 || summary.LeftMinor != tc.want-1500 {
						t.Errorf("unexpected Other spending remainder: %+v", summary)
					}
					return
				}
			}
			t.Fatal("missing Other spending row")
		})
	}
}
