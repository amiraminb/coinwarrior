package internal

import (
	"testing"
	"time"

	"github.com/amiraminb/coinwarrior/internal/domain"
)

func TestSetMonthlyBudgetCreatesAndUpdates(t *testing.T) {
	setupBudgetTestData(t, nil, nil, nil)

	now := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	budget, err := setMonthlyBudgetWithNow("2026-04", "cad", "2000", nil, now)
	if err != nil {
		t.Fatalf("setMonthlyBudgetWithNow returned error: %v", err)
	}
	if budget.Month != "2026-04" || budget.Currency != "CAD" || budget.AmountMinor != 200000 {
		t.Fatalf("unexpected budget created: %+v", budget)
	}

	updated, err := setMonthlyBudgetWithNow("2026-04", "CAD", "2500", nil, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("setMonthlyBudgetWithNow update returned error: %v", err)
	}
	if updated.AmountMinor != 250000 {
		t.Fatalf("expected updated amount 250000, got %d", updated.AmountMinor)
	}

	budgetsFile := loadBudgetTestFile(t)
	if len(budgetsFile.Budgets) != 1 {
		t.Fatalf("expected 1 budget, got %d", len(budgetsFile.Budgets))
	}
	if budgetsFile.Budgets[0].AmountMinor != 250000 {
		t.Fatalf("expected stored amount 250000, got %d", budgetsFile.Budgets[0].AmountMinor)
	}
}

func TestGetMonthlyBudgetSummariesSeparatesCurrencies(t *testing.T) {
	setupBudgetTestData(
		t,
		nil,
		[]domain.Transaction{
			{ID: "cad-exp", Type: TransactionTypeExpense, AmountMinor: 3000, Currency: "CAD", Date: "2026-04-05"},
			{ID: "usd-exp", Type: TransactionTypeExpense, AmountMinor: 1000, Currency: "USD", Date: "2026-04-06"},
			{ID: "cad-income", Type: TransactionTypeIncome, AmountMinor: 9000, Currency: "CAD", Date: "2026-04-02"},
			{ID: "may-exp", Type: TransactionTypeExpense, AmountMinor: 500, Currency: "CAD", Date: "2026-05-01"},
		},
		[]domain.Budget{
			{Month: "2026-04", Currency: "CAD", AmountMinor: 10000, UpdatedAt: "2026-04-01T00:00:00Z"},
			{Month: "2026-04", Currency: "USD", AmountMinor: 5000, UpdatedAt: "2026-04-01T00:00:00Z"},
		},
	)

	summaries, err := GetMonthlyBudgetSummaries("2026-04", time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetMonthlyBudgetSummaries returned error: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	if summaries[0].Budget.Currency != "CAD" || summaries[0].SpentMinor != 3000 || summaries[0].LeftMinor != 7000 {
		t.Fatalf("unexpected CAD summary: %+v", summaries[0])
	}
	if summaries[1].Budget.Currency != "USD" || summaries[1].SpentMinor != 1000 || summaries[1].LeftMinor != 4000 {
		t.Fatalf("unexpected USD summary: %+v", summaries[1])
	}
}

func TestGetBudgetCarryoverCandidateReturnsPreviousMonthLeft(t *testing.T) {
	setupBudgetTestData(
		t,
		nil,
		[]domain.Transaction{{ID: "apr-exp", Type: TransactionTypeExpense, AmountMinor: 2500, Currency: "CAD", Date: "2026-04-10"}},
		[]domain.Budget{{Month: "2026-04", Currency: "CAD", AmountMinor: 10000, UpdatedAt: "2026-04-01T00:00:00Z"}},
	)

	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	candidate, err := GetBudgetCarryoverCandidate("2026-05", "CAD", now)
	if err != nil {
		t.Fatalf("GetBudgetCarryoverCandidate returned error: %v", err)
	}
	if candidate == nil {
		t.Fatal("expected carryover candidate, got nil")
	}
	if candidate.SourceBudget.Month != "2026-04" || candidate.TargetMonth != "2026-05" || candidate.LeftMinor != 7500 {
		t.Fatalf("unexpected carryover candidate: %+v", *candidate)
	}
}

func TestSetMonthlyBudgetWithCarryoverCarriesPreviousLeft(t *testing.T) {
	setupBudgetTestData(
		t,
		nil,
		[]domain.Transaction{{ID: "apr-exp", Type: TransactionTypeExpense, AmountMinor: 2500, Currency: "CAD", Date: "2026-04-10"}},
		[]domain.Budget{{Month: "2026-04", Currency: "CAD", AmountMinor: 10000, UpdatedAt: "2026-04-01T00:00:00Z"}},
	)

	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	carry := true
	budget, err := setMonthlyBudgetWithNow("2026-05", "CAD", "2000", &carry, now)
	if err != nil {
		t.Fatalf("setMonthlyBudgetWithNow returned error: %v", err)
	}
	if budget.Month != "2026-05" || budget.AmountMinor != 200000 || budget.RolloverMinor != 7500 || budget.RolloverFromMonth != "2026-04" {
		t.Fatalf("unexpected target budget: %+v", budget)
	}

	budgetsFile := loadBudgetTestFile(t)
	if len(budgetsFile.Budgets) != 2 {
		t.Fatalf("expected 2 budgets, got %d", len(budgetsFile.Budgets))
	}
	if budgetsFile.Budgets[0].RolloverStatus != budgetRolloverStatusCarried || budgetsFile.Budgets[0].RolledOverIntoMonth != "2026-05" {
		t.Fatalf("unexpected source budget after carryover: %+v", budgetsFile.Budgets[0])
	}
}

func TestSetMonthlyBudgetWithCarryoverSkipMarksPreviousMonth(t *testing.T) {
	setupBudgetTestData(
		t,
		nil,
		[]domain.Transaction{{ID: "apr-exp", Type: TransactionTypeExpense, AmountMinor: 2500, Currency: "CAD", Date: "2026-04-10"}},
		[]domain.Budget{{Month: "2026-04", Currency: "CAD", AmountMinor: 10000, UpdatedAt: "2026-04-01T00:00:00Z"}},
	)

	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	carry := false
	budget, err := setMonthlyBudgetWithNow("2026-05", "CAD", "2000", &carry, now)
	if err != nil {
		t.Fatalf("setMonthlyBudgetWithNow returned error: %v", err)
	}
	if budget.RolloverMinor != 0 || budget.RolloverFromMonth != "" {
		t.Fatalf("expected no rollover on target budget, got %+v", budget)
	}

	budgetsFile := loadBudgetTestFile(t)
	if budgetsFile.Budgets[0].RolloverStatus != budgetRolloverStatusSkipped {
		t.Fatalf("unexpected source budget after skip: %+v", budgetsFile.Budgets[0])
	}
}

func TestApplyMonthlyBudgetRolloverCarriesSignedLeft(t *testing.T) {
	setupBudgetTestData(
		t,
		nil,
		[]domain.Transaction{{ID: "apr-exp", Type: TransactionTypeExpense, AmountMinor: 13000, Currency: "CAD", Date: "2026-04-10"}},
		[]domain.Budget{{Month: "2026-04", Currency: "CAD", AmountMinor: 10000, UpdatedAt: "2026-04-01T00:00:00Z"}},
	)

	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	source, destination, err := ApplyMonthlyBudgetRollover("2026-04", "CAD", true, now)
	if err != nil {
		t.Fatalf("ApplyMonthlyBudgetRollover returned error: %v", err)
	}
	if source.RolloverStatus != budgetRolloverStatusCarried || source.RolledOverMinor != -3000 || source.RolledOverIntoMonth != "2026-05" {
		t.Fatalf("unexpected source budget after rollover: %+v", source)
	}
	if destination == nil {
		t.Fatal("expected destination budget, got nil")
	}
	if destination.Month != "2026-05" || destination.RolloverMinor != -3000 || destination.RolloverFromMonth != "2026-04" {
		t.Fatalf("unexpected destination budget: %+v", *destination)
	}

	budgetsFile := loadBudgetTestFile(t)
	if len(budgetsFile.Budgets) != 2 {
		t.Fatalf("expected 2 budgets after rollover, got %d", len(budgetsFile.Budgets))
	}
}

func TestApplyMonthlyBudgetRolloverSkipMarksBudget(t *testing.T) {
	setupBudgetTestData(
		t,
		nil,
		[]domain.Transaction{{ID: "apr-exp", Type: TransactionTypeExpense, AmountMinor: 2500, Currency: "CAD", Date: "2026-04-10"}},
		[]domain.Budget{{Month: "2026-04", Currency: "CAD", AmountMinor: 10000, UpdatedAt: "2026-04-01T00:00:00Z"}},
	)

	now := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
	source, destination, err := ApplyMonthlyBudgetRollover("2026-04", "CAD", false, now)
	if err != nil {
		t.Fatalf("ApplyMonthlyBudgetRollover returned error: %v", err)
	}
	if destination != nil {
		t.Fatalf("expected no destination budget, got %+v", destination)
	}
	if source.RolloverStatus != budgetRolloverStatusSkipped || source.RolledOverIntoMonth != "" {
		t.Fatalf("unexpected skipped source budget: %+v", source)
	}

	budgetsFile := loadBudgetTestFile(t)
	if len(budgetsFile.Budgets) != 1 {
		t.Fatalf("expected 1 budget after skipped rollover, got %d", len(budgetsFile.Budgets))
	}
	if budgetsFile.Budgets[0].RolloverStatus != budgetRolloverStatusSkipped {
		t.Fatalf("expected stored skipped status, got %+v", budgetsFile.Budgets[0])
	}
}

func setupBudgetTestData(t *testing.T, accounts []domain.Account, transactions []domain.Transaction, budgets []domain.Budget) {
	t.Helper()

	setupTransactionTestData(t, accounts, transactions)

	path, err := FilePath(BudgetsFileName)
	if err != nil {
		t.Fatalf("FilePath returned error: %v", err)
	}
	if err := SaveBudgetsFile(path, domain.BudgetsFile{SchemaVersion: 1, Budgets: budgets}); err != nil {
		t.Fatalf("SaveBudgetsFile returned error: %v", err)
	}
}

func loadBudgetTestFile(t *testing.T) domain.BudgetsFile {
	t.Helper()

	path, err := FilePath(BudgetsFileName)
	if err != nil {
		t.Fatalf("FilePath returned error: %v", err)
	}

	budgetsFile, err := LoadBudgetsFile(path)
	if err != nil {
		t.Fatalf("LoadBudgetsFile returned error: %v", err)
	}

	return budgetsFile
}
