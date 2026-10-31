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
	verboseFlag := reportOverviewCmd.Flags().Lookup("verbose")
	previousVerbose, previousChanged := verboseFlag.Value.String(), verboseFlag.Changed
	t.Cleanup(func() {
		if err := reportOverviewCmd.Flags().Set("verbose", previousVerbose); err != nil {
			t.Error(err)
		}
		verboseFlag.Changed = previousChanged
	})
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
			name:       "verbose filtered month with budgets",
			args:       []string{"groceries", date("01-01") + ".." + date("01-31"), "-v"},
			want:       []string{"Groceries", "Monthly Budget", "500.00", "-250.00", "25.0%"},
			absent:     []string{"Housing", "Unused", "All spending", "Per-Month Breakdown"},
			tableCount: 1,
		},
		{
			name:       "category without transactions",
			args:       []string{"unused", "year", "--verbose"},
			want:       []string{"Unused", "0.00"},
			absent:     []string{"Groceries", "Housing", "Per-Month Breakdown"},
			tableCount: 1,
		},
		{
			name:       "unfiltered year includes saved categories",
			args:       []string{"year"},
			want:       []string{"Groceries", "Housing", "Unused", date("02"), "Monthly Income / Expense"},
			absent:     []string{"Per-Month Breakdown"},
			tableCount: 1,
		},
		{
			name:       "verbose unfiltered year",
			args:       []string{"year", "--verbose"},
			want:       []string{"Groceries", "Housing", "Unused", date("02"), "Per-Month Breakdown"},
			tableCount: 4,
		},
		{
			name:       "verbose filtered year with shorthand",
			args:       []string{"groceries", "year", "-v"},
			want:       []string{"Groceries", "-350.00", "25.0%", "▲ 100.00", "Per-Month Breakdown"},
			absent:     []string{"Housing", "Unused", " groceries ", date("02"), date("04")},
			tableCount: 3,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := reportOverviewCmd.Flags().Set("verbose", "false"); err != nil {
				t.Fatal(err)
			}
			output := captureBudgetOutput(t, func() error {
				if err := reportOverviewCmd.ParseFlags(tc.args); err != nil {
					return err
				}
				return reportOverviewCmd.RunE(reportOverviewCmd, reportOverviewCmd.Flags().Args())
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
