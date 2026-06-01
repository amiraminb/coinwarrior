package service

import (
	"errors"
	"testing"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
)

// fakeRepo is an in-memory Repository for service tests. failRules forces the
// next SaveRecurringRules to fail, exercising the generation rollback path.
type fakeRepo struct {
	transactions     []model.Transaction
	accounts         []model.Account
	rules            []model.RecurringRule
	failRules        bool
	failTransactions bool
}

func (r *fakeRepo) LoadAccounts() ([]model.Account, error) { return cloneAccounts(r.accounts), nil }
func (r *fakeRepo) SaveAccounts(a []model.Account) error   { r.accounts = cloneAccounts(a); return nil }
func (r *fakeRepo) LoadTransactions() ([]model.Transaction, error) {
	return cloneTransactions(r.transactions), nil
}
func (r *fakeRepo) SaveTransactions(t []model.Transaction) error {
	if r.failTransactions {
		return errors.New("forced transactions save failure")
	}
	r.transactions = cloneTransactions(t)
	return nil
}
func (r *fakeRepo) LoadCategories() ([]string, error)    { return nil, nil }
func (r *fakeRepo) SaveCategories([]string) error        { return nil }
func (r *fakeRepo) LoadBudgets() ([]model.Budget, error) { return nil, nil }
func (r *fakeRepo) SaveBudgets([]model.Budget) error     { return nil }
func (r *fakeRepo) LoadRecurringRules() ([]model.RecurringRule, error) {
	out := make([]model.RecurringRule, len(r.rules))
	copy(out, r.rules)
	return out, nil
}
func (r *fakeRepo) SaveRecurringRules(rules []model.RecurringRule) error {
	if r.failRules {
		return errors.New("forced rules save failure")
	}
	out := make([]model.RecurringRule, len(rules))
	copy(out, rules)
	r.rules = out
	return nil
}

func monthlyExpenseRepo(failRules bool) *fakeRepo {
	return &fakeRepo{
		accounts: []model.Account{{Name: "Checking", Currency: "CAD", BalanceMinor: 100000}},
		rules: []model.RecurringRule{{
			ID:          "rec_1",
			Type:        model.TransactionTypeExpense,
			AmountMinor: 5000,
			Currency:    "CAD",
			Category:    "Rent",
			Account:     "Checking",
			DayOfMonth:  1,
			StartDate:   "2026-01-01",
		}},
		failRules: failRules,
	}
}

// March 15 with a Jan 1 start and day-of-month 1 means Jan, Feb, and Mar are due.
var generationNow = time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)

func TestGenerateDueTransactionsIsIdempotent(t *testing.T) {
	repo := monthlyExpenseRepo(false)
	svc := New(repo)

	first, err := svc.GenerateDueTransactions(generationNow)
	if err != nil {
		t.Fatalf("first generation: %v", err)
	}
	if len(first.Generated) != 3 {
		t.Fatalf("first generation: got %d transactions, want 3", len(first.Generated))
	}

	second, err := svc.GenerateDueTransactions(generationNow)
	if err != nil {
		t.Fatalf("second generation: %v", err)
	}
	if len(second.Generated) != 0 {
		t.Fatalf("second generation: got %d transactions, want 0 (idempotent)", len(second.Generated))
	}
	if len(repo.transactions) != 3 {
		t.Fatalf("ledger has %d transactions, want 3", len(repo.transactions))
	}
	if repo.accounts[0].BalanceMinor != 85000 {
		t.Fatalf("balance is %d, want 85000 (100000 - 3*5000)", repo.accounts[0].BalanceMinor)
	}
}

// A rules-save failure must roll back the already-persisted transactions and
// account deltas, so a retry generates each month exactly once rather than
// double-charging.
func TestGenerateDueTransactionsRollsBackOnRulesSaveFailure(t *testing.T) {
	repo := monthlyExpenseRepo(true)
	svc := New(repo)

	if _, err := svc.GenerateDueTransactions(generationNow); err == nil {
		t.Fatal("expected error when rules save fails, got nil")
	}
	if len(repo.transactions) != 0 {
		t.Fatalf("after failed generation: ledger has %d transactions, want 0 (rolled back)", len(repo.transactions))
	}
	if repo.accounts[0].BalanceMinor != 100000 {
		t.Fatalf("after failed generation: balance is %d, want 100000 (rolled back)", repo.accounts[0].BalanceMinor)
	}

	repo.failRules = false
	retry, err := svc.GenerateDueTransactions(generationNow)
	if err != nil {
		t.Fatalf("retry generation: %v", err)
	}
	if len(retry.Generated) != 3 {
		t.Fatalf("retry generation: got %d transactions, want 3", len(retry.Generated))
	}
	if len(repo.transactions) != 3 {
		t.Fatalf("after retry: ledger has %d transactions, want 3 (no double charge)", len(repo.transactions))
	}
	if repo.accounts[0].BalanceMinor != 85000 {
		t.Fatalf("after retry: balance is %d, want 85000 (no double charge)", repo.accounts[0].BalanceMinor)
	}
}

// A transactions-save failure must roll back the account deltas already written,
// leaving balances untouched so the run can be retried cleanly.
func TestGenerateDueTransactionsRollsBackOnTransactionsSaveFailure(t *testing.T) {
	repo := monthlyExpenseRepo(false)
	repo.failTransactions = true
	svc := New(repo)

	if _, err := svc.GenerateDueTransactions(generationNow); err == nil {
		t.Fatal("expected error when transactions save fails, got nil")
	}
	if len(repo.transactions) != 0 {
		t.Fatalf("after failed generation: ledger has %d transactions, want 0", len(repo.transactions))
	}
	if repo.accounts[0].BalanceMinor != 100000 {
		t.Fatalf("after failed generation: balance is %d, want 100000 (rolled back)", repo.accounts[0].BalanceMinor)
	}
	if repo.rules[0].LastGeneratedMonth != "" {
		t.Fatalf("after failed generation: LastGeneratedMonth is %q, want empty (not advanced)", repo.rules[0].LastGeneratedMonth)
	}
}

// ruleRepo builds a single-rule repo for current-month generation tests.
func ruleRepo(rule model.RecurringRule) *fakeRepo {
	return &fakeRepo{
		accounts: []model.Account{{Name: "Checking", Currency: "CAD", BalanceMinor: 100000}},
		rules:    []model.RecurringRule{rule},
	}
}

func expenseRule(day int, start, end string) model.RecurringRule {
	return model.RecurringRule{
		ID:          "rec_1",
		Type:        model.TransactionTypeExpense,
		AmountMinor: 5000,
		Currency:    "CAD",
		Category:    "Rent",
		Account:     "Checking",
		DayOfMonth:  day,
		StartDate:   start,
		EndDate:     end,
	}
}

// The current month's occurrence is generated as soon as the month has begun,
// even when the rule's day-of-month is later in the month.
func TestGenerateDueTransactionsGeneratesCurrentMonthBeforeDueDay(t *testing.T) {
	repo := ruleRepo(expenseRule(28, "2026-06-01", ""))
	svc := New(repo)
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

	result, err := svc.GenerateDueTransactions(now)
	if err != nil {
		t.Fatalf("generation: %v", err)
	}
	if len(result.Generated) != 1 {
		t.Fatalf("got %d transactions, want 1 (current month due now)", len(result.Generated))
	}
	if got := result.Generated[0].Date; got != "2026-06-28" {
		t.Errorf("transaction date = %q, want 2026-06-28 (scheduled day, not generation day)", got)
	}
	if repo.accounts[0].BalanceMinor != 95000 {
		t.Errorf("balance = %d, want 95000 (100000 - 5000)", repo.accounts[0].BalanceMinor)
	}
	if repo.rules[0].LastGeneratedMonth != "2026-06" {
		t.Errorf("LastGeneratedMonth = %q, want 2026-06", repo.rules[0].LastGeneratedMonth)
	}

	// Running again the same month is a no-op.
	second, err := svc.GenerateDueTransactions(now)
	if err != nil {
		t.Fatalf("second generation: %v", err)
	}
	if len(second.Generated) != 0 {
		t.Errorf("second run generated %d, want 0 (idempotent)", len(second.Generated))
	}
}

// Backfilling past months and generating the current month's future-day
// occurrence happen in a single run.
func TestGenerateDueTransactionsBackfillsAndIncludesCurrentMonth(t *testing.T) {
	repo := ruleRepo(expenseRule(28, "2026-04-01", ""))
	svc := New(repo)
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

	result, err := svc.GenerateDueTransactions(now)
	if err != nil {
		t.Fatalf("generation: %v", err)
	}
	dates := make([]string, len(result.Generated))
	for i, tx := range result.Generated {
		dates[i] = tx.Date
	}
	want := []string{"2026-04-28", "2026-05-28", "2026-06-28"}
	if len(dates) != len(want) {
		t.Fatalf("got dates %v, want %v", dates, want)
	}
	for i := range want {
		if dates[i] != want[i] {
			t.Fatalf("got dates %v, want %v", dates, want)
		}
	}
}

// A current-month occurrence past the rule's end date is still excluded.
func TestGenerateDueTransactionsRespectsEndDateInCurrentMonth(t *testing.T) {
	repo := ruleRepo(expenseRule(28, "2026-06-01", "2026-06-15"))
	svc := New(repo)
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

	result, err := svc.GenerateDueTransactions(now)
	if err != nil {
		t.Fatalf("generation: %v", err)
	}
	if len(result.Generated) != 0 {
		t.Errorf("got %d transactions, want 0 (28th is past the 15th end date)", len(result.Generated))
	}
}

// Future months are never generated, even when the current month is.
func TestGenerateDueTransactionsDoesNotGenerateFutureMonths(t *testing.T) {
	repo := ruleRepo(expenseRule(28, "2026-06-01", ""))
	svc := New(repo)
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)

	result, err := svc.GenerateDueTransactions(now)
	if err != nil {
		t.Fatalf("generation: %v", err)
	}
	for _, tx := range result.Generated {
		if tx.Date > "2026-06-30" {
			t.Errorf("generated a future-month transaction dated %q", tx.Date)
		}
	}
	if len(result.Generated) != 1 {
		t.Errorf("got %d transactions, want exactly 1 (current month only)", len(result.Generated))
	}
}
