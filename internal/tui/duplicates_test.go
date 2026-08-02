package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/amiraminb/coinwarrior/internal/service"
)

// A prompt needs a TTY, so a regressed guard would hang or error here rather
// than return true.
func TestConfirmNotDuplicateSkipsThePromptWhenNothingMatches(t *testing.T) {
	previous := Svc
	t.Cleanup(func() { Svc = previous })
	Svc = service.New(&duplicateStubRepo{})

	proceed, err := confirmNotDuplicate(model.TransactionTypeExpense, "20.00", "CAD", "2026-08-01", "Groceries")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !proceed {
		t.Error("an unmatched transaction should proceed without a prompt")
	}
}

func TestConfirmNotDuplicateSkipsThePromptForTransfers(t *testing.T) {
	previous := Svc
	t.Cleanup(func() { Svc = previous })
	Svc = service.New(&duplicateStubRepo{transactions: []model.Transaction{{
		Type:        model.TransactionTypeTransfer,
		AmountMinor: 2000,
		Currency:    "CAD",
		Date:        "2026-08-01",
		Category:    model.TransferCategory,
	}}})

	proceed, err := confirmNotDuplicate(model.TransactionTypeTransfer, "20.00", "CAD", "2026-08-01", model.TransferCategory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !proceed {
		t.Error("a transfer should proceed without a prompt")
	}
}

// The add flow passes an empty category for a transfer, so the transfer guard
// has to short-circuit before the category is ever compared.
func TestConfirmNotDuplicateHandlesATransferWithNoCategory(t *testing.T) {
	previous := Svc
	t.Cleanup(func() { Svc = previous })
	Svc = service.New(&duplicateStubRepo{transactions: []model.Transaction{{
		Type:        model.TransactionTypeExpense,
		AmountMinor: 2000,
		Currency:    "CAD",
		Date:        "2026-08-01",
	}}})

	proceed, err := confirmNotDuplicate(model.TransactionTypeTransfer, "20.00", "CAD", "2026-08-01", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !proceed {
		t.Error("a transfer with no category should proceed without a prompt")
	}
}

func TestConfirmNotDuplicateSurfacesAnUnparseableAmount(t *testing.T) {
	previous := Svc
	t.Cleanup(func() { Svc = previous })
	Svc = service.New(&duplicateStubRepo{})

	proceed, err := confirmNotDuplicate(model.TransactionTypeExpense, "not-a-number", "CAD", "2026-08-01", "Groceries")
	if err == nil {
		t.Fatal("expected an error for an unparseable amount")
	}
	if proceed {
		t.Error("must not proceed when the amount cannot be parsed")
	}
}

func TestConfirmNotDuplicateSurfacesALoadFailure(t *testing.T) {
	previous := Svc
	t.Cleanup(func() { Svc = previous })
	Svc = service.New(&duplicateStubRepo{loadErr: errors.New("boom")})

	proceed, err := confirmNotDuplicate(model.TransactionTypeExpense, "20.00", "CAD", "2026-08-01", "Groceries")
	if err == nil {
		t.Fatal("expected the load failure to surface")
	}
	if proceed {
		t.Error("a load failure must not report that the save should proceed")
	}
}

// Implements every repository method explicitly rather than embedding the
// interface, so a new method breaks the build instead of panicking at runtime.
type duplicateStubRepo struct {
	transactions []model.Transaction
	loadErr      error
}

var _ repository.Repository = (*duplicateStubRepo)(nil)

func (r *duplicateStubRepo) LoadTransactions() ([]model.Transaction, error) {
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	return r.transactions, nil
}

func (r *duplicateStubRepo) SaveTransactions([]model.Transaction) error         { return nil }
func (r *duplicateStubRepo) LoadAccounts() ([]model.Account, error)             { return nil, nil }
func (r *duplicateStubRepo) SaveAccounts([]model.Account) error                 { return nil }
func (r *duplicateStubRepo) LoadCategories() ([]string, error)                  { return nil, nil }
func (r *duplicateStubRepo) SaveCategories([]string) error                      { return nil }
func (r *duplicateStubRepo) LoadBudgets() ([]model.Budget, error)               { return nil, nil }
func (r *duplicateStubRepo) SaveBudgets([]model.Budget) error                   { return nil }
func (r *duplicateStubRepo) LoadRecurringRules() ([]model.RecurringRule, error) { return nil, nil }
func (r *duplicateStubRepo) SaveRecurringRules([]model.RecurringRule) error     { return nil }

func TestDuplicateWarningDescribesTheMatches(t *testing.T) {
	match := model.Transaction{
		ID:          "txn_1",
		Type:        model.TransactionTypeExpense,
		AmountMinor: 2000,
		Currency:    "CAD",
		Date:        "2026-08-01",
		Category:    "Groceries",
		Account:     "Chk",
		Note:        "loblaws",
	}

	got := duplicateWarning([]model.Transaction{match}, "20.00", "CAD", "2026-08-01", "Groceries")

	for _, want := range []string{
		"might be a duplicate",
		"1 existing transaction already matches",
		"anyway?",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("warning does not mention %q:\n%s", want, got)
		}
	}

	// The listed match must use the shared formatter, so duplicate warnings show
	// signed amounts and accounts exactly like the delete confirmation does.
	if !strings.Contains(got, FormatEditableTransaction(match)) {
		t.Errorf("warning does not render the match via FormatEditableTransaction:\n%s", got)
	}
}

func TestDuplicateWarningPluralizesMultipleMatches(t *testing.T) {
	match := model.Transaction{
		Type:        model.TransactionTypeExpense,
		AmountMinor: 2000,
		Currency:    "CAD",
		Date:        "2026-08-01",
		Category:    "Groceries",
	}

	got := duplicateWarning([]model.Transaction{match, match}, "20.00", "CAD", "2026-08-01", "Groceries")

	if !strings.Contains(got, "2 existing transactions") {
		t.Errorf("expected a pluralized count for two matches:\n%s", got)
	}
}
