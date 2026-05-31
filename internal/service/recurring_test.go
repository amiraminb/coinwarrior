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
