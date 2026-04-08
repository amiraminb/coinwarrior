package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
)

func TestEditTransactionUpdatesExpenseBalance(t *testing.T) {
	setupTransactionTestData(t,
		[]model.Account{{Name: "Checking", Currency: "CAD", BalanceMinor: 90000, UpdatedAt: "2026-04-01T00:00:00Z"}},
		[]model.Transaction{{
			ID:          "tx1",
			Type:        TransactionTypeExpense,
			AmountMinor: 10000,
			Currency:    "CAD",
			Date:        "2026-04-01",
			Category:    "Groceries",
			Account:     "Checking",
			Note:        "old note",
			CreatedAt:   "2026-04-01T00:00:00Z",
			UpdatedAt:   "2026-04-01T00:00:00Z",
			Source:      "manual",
		}},
	)

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	amount := "150"

	tx, err := editTransactionWithNow("tx1", TransactionEdits{Amount: &amount}, now)
	if err != nil {
		t.Fatalf("editTransactionWithNow returned error: %v", err)
	}
	if tx.AmountMinor != 15000 {
		t.Fatalf("expected updated amount 15000, got %d", tx.AmountMinor)
	}
	if tx.UpdatedAt != now.Format(time.RFC3339) {
		t.Fatalf("expected updated timestamp %s, got %s", now.Format(time.RFC3339), tx.UpdatedAt)
	}

	accountsFile, transactionsFile := loadTransactionTestState(t)
	if got := accountBalance(t, accountsFile.Accounts, "Checking"); got != 85000 {
		t.Fatalf("expected checking balance 85000, got %d", got)
	}
	if got := transactionsFile.Transactions[0].AmountMinor; got != 15000 {
		t.Fatalf("expected saved amount 15000, got %d", got)
	}
}

func TestEditTransactionUpdatesTransferBalances(t *testing.T) {
	setupTransactionTestData(t,
		[]model.Account{
			{Name: "Checking", Currency: "CAD", BalanceMinor: 90000, UpdatedAt: "2026-04-01T00:00:00Z"},
			{Name: "Travel", Currency: "CAD", BalanceMinor: 110000, UpdatedAt: "2026-04-01T00:00:00Z"},
			{Name: "Savings", Currency: "CAD", BalanceMinor: 50000, UpdatedAt: "2026-04-01T00:00:00Z"},
		},
		[]model.Transaction{{
			ID:          "tx-transfer",
			Type:        TransactionTypeTransfer,
			AmountMinor: 10000,
			Currency:    "CAD",
			Date:        "2026-04-01",
			Category:    "Transfer",
			Account:     "Checking",
			ToAccount:   "Travel",
			CreatedAt:   "2026-04-01T00:00:00Z",
			UpdatedAt:   "2026-04-01T00:00:00Z",
			Source:      "manual",
		}},
	)

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	amount := "70"
	account := "Savings"
	toAccount := "Checking"

	tx, err := editTransactionWithNow("tx-transfer", TransactionEdits{Amount: &amount, Account: &account, ToAccount: &toAccount}, now)
	if err != nil {
		t.Fatalf("editTransactionWithNow returned error: %v", err)
	}
	if tx.AmountMinor != 7000 {
		t.Fatalf("expected updated amount 7000, got %d", tx.AmountMinor)
	}
	if tx.Account != "Savings" || tx.ToAccount != "Checking" {
		t.Fatalf("expected Savings -> Checking, got %s -> %s", tx.Account, tx.ToAccount)
	}

	accountsFile, _ := loadTransactionTestState(t)
	if got := accountBalance(t, accountsFile.Accounts, "Checking"); got != 107000 {
		t.Fatalf("expected checking balance 107000, got %d", got)
	}
	if got := accountBalance(t, accountsFile.Accounts, "Travel"); got != 100000 {
		t.Fatalf("expected travel balance 100000, got %d", got)
	}
	if got := accountBalance(t, accountsFile.Accounts, "Savings"); got != 43000 {
		t.Fatalf("expected savings balance 43000, got %d", got)
	}
}

func TestEditTransactionMetadataDoesNotChangeBalance(t *testing.T) {
	setupTransactionTestData(t,
		[]model.Account{{Name: "Checking", Currency: "CAD", BalanceMinor: 120000, UpdatedAt: "2026-04-01T00:00:00Z"}},
		[]model.Transaction{{
			ID:          "tx2",
			Type:        TransactionTypeIncome,
			AmountMinor: 20000,
			Currency:    "CAD",
			Date:        "2026-04-01",
			Category:    "Income",
			Account:     "Checking",
			Note:        "April pay",
			CreatedAt:   "2026-04-01T00:00:00Z",
			UpdatedAt:   "2026-04-01T00:00:00Z",
			Source:      "manual",
		}},
	)

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	date := "2026-04-02"
	category := "Salary"
	note := "Updated note"

	tx, err := editTransactionWithNow("tx2", TransactionEdits{Date: &date, Category: &category, Note: &note}, now)
	if err != nil {
		t.Fatalf("editTransactionWithNow returned error: %v", err)
	}
	if tx.Date != date || tx.Category != category || tx.Note != note {
		t.Fatalf("transaction fields not updated correctly: %+v", tx)
	}

	accountsFile, _ := loadTransactionTestState(t)
	if got := accountBalance(t, accountsFile.Accounts, "Checking"); got != 120000 {
		t.Fatalf("expected checking balance 120000, got %d", got)
	}
}

func TestEditTransactionRollbackOnInvalidAccountChange(t *testing.T) {
	setupTransactionTestData(t,
		[]model.Account{
			{Name: "Checking", Currency: "CAD", BalanceMinor: 90000, UpdatedAt: "2026-04-01T00:00:00Z"},
			{Name: "USD Wallet", Currency: "USD", BalanceMinor: 50000, UpdatedAt: "2026-04-01T00:00:00Z"},
		},
		[]model.Transaction{{
			ID:          "tx3",
			Type:        TransactionTypeExpense,
			AmountMinor: 10000,
			Currency:    "CAD",
			Date:        "2026-04-01",
			Category:    "Groceries",
			Account:     "Checking",
			CreatedAt:   "2026-04-01T00:00:00Z",
			UpdatedAt:   "2026-04-01T00:00:00Z",
			Source:      "manual",
		}},
	)

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	account := "USD Wallet"

	_, err := editTransactionWithNow("tx3", TransactionEdits{Account: &account}, now)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "uses currency USD") {
		t.Fatalf("expected currency validation error, got %v", err)
	}

	accountsFile, transactionsFile := loadTransactionTestState(t)
	if got := accountBalance(t, accountsFile.Accounts, "Checking"); got != 90000 {
		t.Fatalf("expected checking balance 90000 after rollback, got %d", got)
	}
	if got := transactionsFile.Transactions[0].Account; got != "Checking" {
		t.Fatalf("expected transaction account to remain Checking, got %s", got)
	}
}

func TestDeleteTransactionReversesExpenseBalance(t *testing.T) {
	setupTransactionTestData(t,
		[]model.Account{{Name: "Checking", Currency: "CAD", BalanceMinor: 90000, UpdatedAt: "2026-04-01T00:00:00Z"}},
		[]model.Transaction{{
			ID:          "tx-delete-expense",
			Type:        TransactionTypeExpense,
			AmountMinor: 10000,
			Currency:    "CAD",
			Date:        "2026-04-01",
			Category:    "Groceries",
			Account:     "Checking",
			CreatedAt:   "2026-04-01T00:00:00Z",
			UpdatedAt:   "2026-04-01T00:00:00Z",
			Source:      "manual",
		}},
	)

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	tx, err := deleteTransactionWithNow("tx-delete-expense", now)
	if err != nil {
		t.Fatalf("deleteTransactionWithNow returned error: %v", err)
	}
	if tx.ID != "tx-delete-expense" {
		t.Fatalf("expected deleted transaction id tx-delete-expense, got %s", tx.ID)
	}

	accountsFile, transactionsFile := loadTransactionTestState(t)
	if got := accountBalance(t, accountsFile.Accounts, "Checking"); got != 100000 {
		t.Fatalf("expected checking balance 100000, got %d", got)
	}
	if len(transactionsFile.Transactions) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(transactionsFile.Transactions))
	}
}

func TestDeleteTransactionReversesIncomeBalance(t *testing.T) {
	setupTransactionTestData(t,
		[]model.Account{{Name: "Checking", Currency: "CAD", BalanceMinor: 120000, UpdatedAt: "2026-04-01T00:00:00Z"}},
		[]model.Transaction{{
			ID:          "tx-delete-income",
			Type:        TransactionTypeIncome,
			AmountMinor: 20000,
			Currency:    "CAD",
			Date:        "2026-04-01",
			Category:    "Income",
			Account:     "Checking",
			CreatedAt:   "2026-04-01T00:00:00Z",
			UpdatedAt:   "2026-04-01T00:00:00Z",
			Source:      "manual",
		}},
	)

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	_, err := deleteTransactionWithNow("tx-delete-income", now)
	if err != nil {
		t.Fatalf("deleteTransactionWithNow returned error: %v", err)
	}

	accountsFile, transactionsFile := loadTransactionTestState(t)
	if got := accountBalance(t, accountsFile.Accounts, "Checking"); got != 100000 {
		t.Fatalf("expected checking balance 100000, got %d", got)
	}
	if len(transactionsFile.Transactions) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(transactionsFile.Transactions))
	}
}

func TestDeleteTransactionReversesTransferBalances(t *testing.T) {
	setupTransactionTestData(t,
		[]model.Account{
			{Name: "Checking", Currency: "CAD", BalanceMinor: 90000, UpdatedAt: "2026-04-01T00:00:00Z"},
			{Name: "Travel", Currency: "CAD", BalanceMinor: 110000, UpdatedAt: "2026-04-01T00:00:00Z"},
		},
		[]model.Transaction{{
			ID:          "tx-delete-transfer",
			Type:        TransactionTypeTransfer,
			AmountMinor: 10000,
			Currency:    "CAD",
			Date:        "2026-04-01",
			Category:    "Transfer",
			Account:     "Checking",
			ToAccount:   "Travel",
			CreatedAt:   "2026-04-01T00:00:00Z",
			UpdatedAt:   "2026-04-01T00:00:00Z",
			Source:      "manual",
		}},
	)

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	_, err := deleteTransactionWithNow("tx-delete-transfer", now)
	if err != nil {
		t.Fatalf("deleteTransactionWithNow returned error: %v", err)
	}

	accountsFile, transactionsFile := loadTransactionTestState(t)
	if got := accountBalance(t, accountsFile.Accounts, "Checking"); got != 100000 {
		t.Fatalf("expected checking balance 100000, got %d", got)
	}
	if got := accountBalance(t, accountsFile.Accounts, "Travel"); got != 100000 {
		t.Fatalf("expected travel balance 100000, got %d", got)
	}
	if len(transactionsFile.Transactions) != 0 {
		t.Fatalf("expected 0 transactions, got %d", len(transactionsFile.Transactions))
	}
}

func TestDeleteTransactionLeavesFilesUnchangedOnReverseFailure(t *testing.T) {
	setupTransactionTestData(t,
		[]model.Account{{Name: "Savings", Currency: "CAD", BalanceMinor: 50000, UpdatedAt: "2026-04-01T00:00:00Z"}},
		[]model.Transaction{{
			ID:          "tx-delete-fail",
			Type:        TransactionTypeExpense,
			AmountMinor: 10000,
			Currency:    "CAD",
			Date:        "2026-04-01",
			Category:    "Groceries",
			Account:     "Checking",
			CreatedAt:   "2026-04-01T00:00:00Z",
			UpdatedAt:   "2026-04-01T00:00:00Z",
			Source:      "manual",
		}},
	)

	now := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
	_, err := deleteTransactionWithNow("tx-delete-fail", now)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "account 'Checking' not found") {
		t.Fatalf("expected missing account error, got %v", err)
	}

	accountsFile, transactionsFile := loadTransactionTestState(t)
	if got := accountBalance(t, accountsFile.Accounts, "Savings"); got != 50000 {
		t.Fatalf("expected savings balance 50000, got %d", got)
	}
	if len(transactionsFile.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(transactionsFile.Transactions))
	}
	if transactionsFile.Transactions[0].ID != "tx-delete-fail" {
		t.Fatalf("expected transaction to remain, got %s", transactionsFile.Transactions[0].ID)
	}
}

func setupTransactionTestData(t *testing.T, accounts []model.Account, transactions []model.Transaction) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	dataDir := filepath.Join(home, "Documents", ".coinwarrior")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	if err := SaveAccountsFile(filepath.Join(dataDir, AccountsFileName), model.AccountsFile{SchemaVersion: 1, Accounts: accounts}); err != nil {
		t.Fatalf("SaveAccountsFile returned error: %v", err)
	}
	if err := SaveTransactions(filepath.Join(dataDir, TransactionsFileName), model.TransactionsFile{SchemaVersion: 1, Transactions: transactions}); err != nil {
		t.Fatalf("SaveTransactions returned error: %v", err)
	}
}

func loadTransactionTestState(t *testing.T) (model.AccountsFile, model.TransactionsFile) {
	t.Helper()

	accountsPath, err := FilePath(AccountsFileName)
	if err != nil {
		t.Fatalf("FilePath returned error: %v", err)
	}
	transactionsPath, err := FilePath(TransactionsFileName)
	if err != nil {
		t.Fatalf("FilePath returned error: %v", err)
	}

	accountsFile, err := LoadAccountsFile(accountsPath)
	if err != nil {
		t.Fatalf("LoadAccountsFile returned error: %v", err)
	}
	transactionsFile, err := LoadTransactions(transactionsPath)
	if err != nil {
		t.Fatalf("LoadTransactions returned error: %v", err)
	}

	return accountsFile, transactionsFile
}

func accountBalance(t *testing.T, accounts []model.Account, name string) int64 {
	t.Helper()

	for _, account := range accounts {
		if account.Name == name {
			return account.BalanceMinor
		}
	}

	t.Fatalf("account %s not found", name)
	return 0
}
