package cmd

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/amiraminb/coinwarrior/internal/service"
)

func TestOverviewCategoryFilterAppliesToEveryTable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := repository.NewFileRepository()
	previousRepo, previousService := repo, svc
	repo, svc = r, service.New(r)
	t.Cleanup(func() { repo, svc = previousRepo, previousService })
	if err := r.SaveCategories([]string{"Groceries", "Housing", "Unused"}); err != nil {
		t.Fatal(err)
	}
	year := time.Now().Year()
	date := func(suffix string) string { return fmt.Sprintf("%d-%s", year, suffix) }
	if err := r.SaveTransactions([]model.Transaction{
		{Type: "expense", Date: date("01-05"), Category: "Groceries", Currency: "CAD", AmountMinor: 25000},
		{Type: "expense", Date: date("03-05"), Category: " groceries ", Currency: "CAD", AmountMinor: 10000},
		{Type: "expense", Date: date("01-10"), Category: "Housing", Currency: "CAD", AmountMinor: 75000},
		{Type: "expense", Date: date("02-10"), Category: "Housing", Currency: "CAD", AmountMinor: 12000},
	}); err != nil {
		t.Fatal(err)
	}
	if err := r.SaveBudgets([]model.Budget{
		{Month: date("01"), Currency: "CAD", AmountMinor: 150000},
		{Month: date("01"), Currency: "CAD", Category: "Groceries", AmountMinor: 50000},
		{Month: date("01"), Currency: "CAD", Category: "Housing", AmountMinor: 100000},
	}); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name       string
		args       []string
		want       []string
		absent     []string
		tableCount int
	}{
		{
			name:       "filtered year",
			args:       []string{"groceries", "year"},
			want:       []string{"Groceries", "-350.00", "28.7%", "Monthly Income / Expense", "Spending Future Value"},
			absent:     []string{"Housing", "Unused", " groceries ", date("02"), date("04"), "Per-Month Breakdown"},
			tableCount: 1,
		},
		{
			name:       "filtered month with budgets",
			args:       []string{"groceries", date("01-01") + ".." + date("01-31")},
			want:       []string{"Groceries", "Monthly Budget", "500.00", "-250.00", "25.0%"},
			absent:     []string{"Housing", "Unused", "All spending", "Per-Month Breakdown"},
			tableCount: 1,
		},
		{
			name:       "category without transactions",
			args:       []string{"unused", "year"},
			want:       []string{"Unused", "0.00"},
			absent:     []string{"Groceries", "Housing", "Per-Month Breakdown"},
			tableCount: 1,
		},
		{
			name:       "unfiltered year includes saved categories",
			args:       []string{"year"},
			want:       []string{"Groceries", "Housing", "Unused", date("02"), "Per-Month Breakdown"},
			tableCount: 4,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := captureBudgetOutput(t, func() error {
				return reportOverviewCmd.RunE(reportOverviewCmd, tc.args)
			})
			for _, want := range tc.want {
				if !strings.Contains(output, want) {
					t.Errorf("overview omitted %q:\n%s", want, output)
				}
			}
			for _, absent := range tc.absent {
				if strings.Contains(output, absent) {
					t.Errorf("overview unexpectedly includes %q:\n%s", absent, output)
				}
			}
			if count := strings.Count(output, "Category Totals"); count != tc.tableCount {
				t.Errorf("got %d category tables, want %d", count, tc.tableCount)
			}
		})
	}
}
