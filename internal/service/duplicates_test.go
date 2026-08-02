package service

import (
	"testing"

	"github.com/amiraminb/coinwarrior/internal/model"
)

func duplicatesRepo(transactions ...model.Transaction) *fakeRepo {
	return &fakeRepo{
		accounts:     []model.Account{{Name: "Chk", Currency: "CAD", BalanceMinor: 1000000}},
		transactions: transactions,
	}
}

func expenseTx(id, date, category string, amountMinor int64) model.Transaction {
	return model.Transaction{
		ID:          id,
		Type:        model.TransactionTypeExpense,
		AmountMinor: amountMinor,
		Currency:    "CAD",
		Date:        date,
		Category:    category,
		Account:     "Chk",
	}
}

func TestFindPossibleDuplicatesMatchesOnDateCategoryAmountAndType(t *testing.T) {
	svc := New(duplicatesRepo(expenseTx("t1", "2026-08-01", "Groceries", 2000)))

	matches, err := svc.FindPossibleDuplicates(model.TransactionTypeExpense, "20.00", "2026-08-01", "Groceries")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != "t1" {
		t.Fatalf("got %d matches (%v), want exactly t1", len(matches), matches)
	}
}

func TestFindPossibleDuplicatesIgnoresNonMatches(t *testing.T) {
	existing := expenseTx("t1", "2026-08-01", "Groceries", 2000)
	incomeSameAmount := expenseTx("t2", "2026-08-01", "Groceries", 2000)
	incomeSameAmount.Type = model.TransactionTypeIncome
	transfer := expenseTx("t3", "2026-08-01", "Groceries", 2000)
	transfer.Type = model.TransactionTypeTransfer

	svc := New(duplicatesRepo(
		existing,
		incomeSameAmount,
		transfer,
		expenseTx("t4", "2026-08-02", "Groceries", 2000),
		expenseTx("t5", "2026-08-01", "Dining", 2000),
		expenseTx("t6", "2026-08-01", "Groceries", 2001),
	))

	matches, err := svc.FindPossibleDuplicates(model.TransactionTypeExpense, "20.00", "2026-08-01", "Groceries")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 || matches[0].ID != "t1" {
		t.Fatalf("got %d matches (%v), want only t1", len(matches), matches)
	}
}

func TestFindPossibleDuplicatesMatchesCategoryCaseInsensitively(t *testing.T) {
	svc := New(duplicatesRepo(expenseTx("t1", "2026-08-01", "Groceries", 2000)))

	matches, err := svc.FindPossibleDuplicates(model.TransactionTypeExpense, "20.00", "2026-08-01", "groceries")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("got %d matches, want 1 despite differing case", len(matches))
	}
}

func TestFindPossibleDuplicatesReturnsEveryMatch(t *testing.T) {
	svc := New(duplicatesRepo(
		expenseTx("t1", "2026-08-01", "Groceries", 2000),
		expenseTx("t2", "2026-08-01", "Groceries", 2000),
	))

	matches, err := svc.FindPossibleDuplicates(model.TransactionTypeExpense, "20.00", "2026-08-01", "Groceries")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("got %d matches, want 2", len(matches))
	}
}

// Two transfers that genuinely match on every key must still not be reported,
// so this pins the early return rather than the in-loop transfer skip.
func TestFindPossibleDuplicatesNeverFlagsATransferCandidate(t *testing.T) {
	existingTransfer := expenseTx("t1", "2026-08-01", model.TransferCategory, 2000)
	existingTransfer.Type = model.TransactionTypeTransfer
	svc := New(duplicatesRepo(existingTransfer))

	matches, err := svc.FindPossibleDuplicates(model.TransactionTypeTransfer, "20.00", "2026-08-01", model.TransferCategory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("got %v, want no matches for a transfer candidate", matches)
	}
}

// Currency is deliberately not a match key, so the same amount in a different
// currency is still flagged. Pins that choice against a silent change.
func TestFindPossibleDuplicatesIgnoresCurrency(t *testing.T) {
	existing := expenseTx("t1", "2026-08-01", "Groceries", 2000)
	existing.Currency = "USD"
	svc := New(duplicatesRepo(existing))

	matches, err := svc.FindPossibleDuplicates(model.TransactionTypeExpense, "20.00", "2026-08-01", "Groceries")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("got %d matches, want 1 despite the differing currency", len(matches))
	}
}

func TestFindPossibleDuplicatesMatchesUncategorizedTransactions(t *testing.T) {
	svc := New(duplicatesRepo(expenseTx("t1", "2026-08-01", "", 2000)))

	matches, err := svc.FindPossibleDuplicates(model.TransactionTypeExpense, "20.00", "2026-08-01", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("got %d matches, want 1 for two uncategorized expenses", len(matches))
	}
}

func TestFindPossibleDuplicatesRejectsAnUnparseableAmount(t *testing.T) {
	svc := New(duplicatesRepo())

	if _, err := svc.FindPossibleDuplicates(model.TransactionTypeExpense, "abc", "2026-08-01", "Groceries"); err == nil {
		t.Fatal("expected an error for an unparseable amount")
	}
}

// A second identical CSV row must warn against the first once it is saved,
// which is what makes the import-flow check catch duplicates within one run.
func TestFindPossibleDuplicatesSeesTransactionsSavedEarlierInTheRun(t *testing.T) {
	repo := duplicatesRepo()
	svc := New(repo)

	matches, err := svc.FindPossibleDuplicates(model.TransactionTypeExpense, "20.00", "2026-08-01", "Groceries")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("got %d matches on an empty ledger, want 0", len(matches))
	}

	if _, err := svc.AddTransaction(model.TransactionTypeExpense, "20.00", "CAD", "2026-08-01", "Groceries", "Chk", "", ""); err != nil {
		t.Fatalf("AddTransaction: %v", err)
	}

	matches, err = svc.FindPossibleDuplicates(model.TransactionTypeExpense, "20.00", "2026-08-01", "Groceries")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("got %d matches after saving an identical transaction, want 1", len(matches))
	}
}
