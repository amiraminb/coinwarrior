package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/amiraminb/coinwarrior/internal/service"
	"github.com/amiraminb/coinwarrior/internal/tui"
)

func TestBudgetCommandsPersistCategoryLimitsAlongsideLegacyBudget(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := repository.NewFileRepository()
	oldService, oldTUIService := svc, tui.Svc
	svc = service.New(r)
	tui.Svc = svc
	t.Cleanup(func() { svc, tui.Svc = oldService, oldTUIService })
	if err := r.SaveBudgets([]model.Budget{{Month: "2026-11", Currency: "CAD", AmountMinor: 350000}}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"set", "Groceries", "2026-11", "500"},
		{"set", "Dining", "2026-11", "450"},
		{"set", "groceries", "2026-11", "550"},
		{"set", "Groceries", "2026-11", "100", "--currency", "USD"},
	} {
		cmd := newBudgetCommand()
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	budgets, err := r.LoadBudgets()
	if err != nil {
		t.Fatal(err)
	}
	if len(budgets) != 4 || budgets[0].Category != "" || budgets[0].AmountMinor != 350000 || budgets[1].AmountMinor != 55000 {
		t.Fatalf("unexpected persisted budgets: %+v", budgets)
	}
	if err := r.SaveTransactions([]model.Transaction{
		{Type: "expense", Date: "2026-11-02", Category: "Groceries", Currency: "CAD", AmountMinor: 2000},
		{Type: "expense", Date: "2026-11-03", Category: "Dining", Currency: "CAD", AmountMinor: 4000},
	}); err != nil {
		t.Fatal(err)
	}
	output := captureBudgetOutput(t, func() error {
		cmd := newBudgetCommand()
		cmd.SetArgs([]string{"show", "2026-11"})
		return cmd.Execute()
	})
	for _, want := range []string{"All spending", "Groceries", "Dining", "550.00", "20.00", "40.00"} {
		if !strings.Contains(output, want) {
			t.Errorf("budget show omitted %q:\n%s", want, output)
		}
	}
	output = captureBudgetOutput(t, func() error {
		return printMonthlyBudgetSection(day(t, "2026-11-01"), day(t, "2026-11-30"), day(t, "2026-11-15"), "Groceries")
	})
	if !strings.Contains(output, "Groceries") || strings.Contains(output, "All spending") || strings.Contains(output, "Dining") {
		t.Errorf("filtered overview must show only its category budget:\n%s", output)
	}
}

func captureBudgetOutput(t *testing.T, run func() error) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "budget-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	previous := os.Stdout
	os.Stdout = f
	defer func() { os.Stdout = previous }()
	if err := run(); err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
