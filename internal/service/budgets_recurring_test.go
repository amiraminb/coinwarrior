package service

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
)

func TestMonthlyBudgetsIncludeRecurringCostsAndRecordedExpenses(t *testing.T) {
	r := &budgetRepo{
		budgets: []model.Budget{
			{Month: "2026-11", Currency: "CAD", Category: "Dining", AmountMinor: 10000},
			{Month: "2026-11", Currency: "EUR", AmountMinor: 20000},
			{Month: "2026-11", Currency: "CAD", AmountMinor: 50000, RolloverMinor: 1000},
			{Month: "2026-10", Currency: "JPY", AmountMinor: 99999},
		},
		categoriesRepo: categoriesRepo{fakeRepo: fakeRepo{
			rules: []model.RecurringRule{
				{ID: "rent", Type: "expense", Currency: "CAD", AmountMinor: 10000, DayOfMonth: 1, StartDate: "2026-01-01", LastGeneratedMonth: "2026-11"},
				{ID: "subscription", Type: "expense", Currency: "cad", AmountMinor: 1500, DayOfMonth: 28, StartDate: "2026-11-01"},
				{ID: "usd", Type: "expense", Currency: "USD", AmountMinor: 2500, DayOfMonth: 5, StartDate: "2026-01-01"},
				{ID: "salary", Type: "income", Currency: "CAD", AmountMinor: 99999, DayOfMonth: 1, StartDate: "2026-01-01"},
				{ID: "transfer", Type: "transfer", Currency: "CAD", AmountMinor: 99999, DayOfMonth: 1, StartDate: "2026-01-01"},
			},
			transactions: []model.Transaction{
				{Type: "expense", Date: "2026-11-01", Currency: "cad", Category: "Housing", AmountMinor: 10000, Source: "recurring"},
				{Type: "expense", Date: "2026-11-30", Currency: "CAD", Category: "Dining", AmountMinor: 500, Source: "recurring"},
				{Type: "expense", Date: "2026-11-10", Currency: "CAD", Category: "Dining", AmountMinor: 700, Source: "manual"},
				{Type: "expense", Date: "2026-11-05", Currency: "USD", AmountMinor: 2600, Source: "recurring"},
				{Type: "expense", Date: "2026-11-05", Currency: "GBP", AmountMinor: 300, Source: "recurring"},
				{Type: "income", Date: "2026-11-01", Currency: "CAD", AmountMinor: 99999, Source: "recurring"},
				{Type: "transfer", Date: "2026-11-01", Currency: "CAD", AmountMinor: 99999, Source: "recurring"},
				{Type: "expense", Date: "2026-10-31", Currency: "CAD", AmountMinor: 99999, Source: "recurring"},
				{Type: "expense", Date: "2026-12-01", Currency: "CAD", AmountMinor: 99999, Source: "recurring"},
			},
		}},
	}
	originalBudgets, originalRules, originalTransactions := slices.Clone(r.budgets), slices.Clone(r.rules), cloneTransactions(r.transactions)
	s := New(r)
	now := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	summaries, err := s.GetMonthlyBudgetSummaries("2026-11", now)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		currency, category   string
		recurring            bool
		planned, spent, left int64
	}{
		{"CAD", "", false, 50000, 11200, 39800},
		{"CAD", "", true, 11500, 10500, 1000},
		{"CAD", "Dining", false, 10000, 1200, 8800},
		{"EUR", "", false, 20000, 0, 20000},
		{"EUR", "", true, 0, 0, 0},
		{"GBP", "", true, 0, 300, -300},
		{"USD", "", true, 2500, 2600, -100},
	}
	if len(summaries) != len(want) {
		t.Fatalf("got %d summaries, want %d: %+v", len(summaries), len(want), summaries)
	}
	for i, expected := range want {
		got := summaries[i]
		if got.Budget.Currency != expected.currency || got.Budget.Category != expected.category || got.Recurring != expected.recurring ||
			got.Budget.AmountMinor != expected.planned || got.SpentMinor != expected.spent || got.LeftMinor != expected.left {
			t.Errorf("row %d = %+v, want %+v", i, got, expected)
		}
		if got.Recurring && (got.Budget.RolloverMinor != 0 || got.Status != "") {
			t.Errorf("recurring costs must not have a rollover or budget status: %+v", got)
		}
	}
	pending, err := s.GetPendingBudgetRollovers("2026-11", now.AddDate(0, 1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 3 {
		t.Errorf("got %d pending rollovers, want only the 3 saved November budgets", len(pending))
	}
	if !reflect.DeepEqual(r.budgets, originalBudgets) || !reflect.DeepEqual(r.rules, originalRules) || !reflect.DeepEqual(r.transactions, originalTransactions) {
		t.Fatal("budget reporting changed saved data")
	}
}

func TestRecurringBudgetRespectsMonthlyOccurrence(t *testing.T) {
	for _, tc := range []struct {
		name, start, end, generated string
		day                         int
		want                        int64
	}{
		{"future day this month", "2026-01-01", "", "", 28, 1000},
		{"already generated", "2026-01-01", "", "2026-12", 1, 1000},
		{"future start", "2026-12-01", "", "", 1, 0},
		{"expired", "2026-01-01", "2026-10-31", "", 1, 0},
		{"start after occurrence", "2026-11-16", "", "", 15, 0},
		{"start on occurrence", "2026-11-15", "", "", 15, 1000},
		{"end before occurrence", "2026-01-01", "2026-11-14", "", 15, 0},
		{"end on occurrence", "2026-01-01", "2026-11-15", "", 15, 1000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, location := range []*time.Location{time.UTC, time.FixedZone("east", 9*60*60), time.FixedZone("west", -8*60*60)} {
				r := &fakeRepo{rules: []model.RecurringRule{{
					ID: "test", Type: "expense", Currency: "CAD", AmountMinor: 1000,
					DayOfMonth: tc.day, StartDate: tc.start, EndDate: tc.end, LastGeneratedMonth: tc.generated,
				}}}
				summaries, err := New(r).GetMonthlyBudgetSummaries("2026-11", time.Date(2026, 11, 1, 0, 0, 0, 0, location))
				if err != nil {
					t.Fatal(err)
				}
				if tc.want == 0 {
					if len(summaries) != 0 {
						t.Errorf("%s: inactive rules should not create rows: %+v", location, summaries)
					}
				} else if len(summaries) != 1 || !summaries[0].Recurring || summaries[0].Budget.AmountMinor != tc.want || summaries[0].LeftMinor != tc.want || summaries[0].SpentMinor != 0 {
					t.Errorf("%s: unexpected scheduled costs: %+v", location, summaries)
				}
			}
		})
	}
}
