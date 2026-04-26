package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/domain"
)

type TransactionEdits struct {
	Date      *string
	Amount    *string
	Category  *string
	Account   *string
	ToAccount *string
	Note      *string
}

func ParseAmount(input string) (int64, error) {
	amount := strings.TrimSpace(input)
	if amount == "" {
		return 0, fmt.Errorf("amount cannot be empty")
	}

	negative := false
	if strings.HasPrefix(amount, "-") {
		negative = true
		amount = strings.TrimSpace(strings.TrimPrefix(amount, "-"))
		if amount == "" {
			return 0, fmt.Errorf("invalid amount: %s", input)
		}
	}

	parts := strings.Split(amount, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid amount format: %s", input)
	}

	whole := parts[0]
	if whole == "" {
		whole = "0"
	}

	wholeValue, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %s", input)
	}

	frac := "00"
	if len(parts) == 2 {
		if len(parts[1]) > 2 {
			return 0, fmt.Errorf("amount supports max 2 decimals: %s", input)
		}
		frac = parts[1]
		if len(frac) == 1 {
			frac += "0"
		}
		if len(frac) == 0 {
			frac = "00"
		}
	}

	fracValue, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %s", input)
	}

	result := wholeValue*100 + fracValue
	if negative {
		result = -result
	}

	return result, nil
}

func NewTransactionID(now time.Time) string {
	return fmt.Sprintf("txn_%d", now.UnixNano())
}

func LoadTransactions(path string) (domain.TransactionsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.TransactionsFile{SchemaVersion: 1, Transactions: []domain.Transaction{}}, nil
		}
		return domain.TransactionsFile{}, err
	}

	var transactions domain.TransactionsFile
	if err := json.Unmarshal(data, &transactions); err != nil {
		return domain.TransactionsFile{}, err
	}
	if transactions.Transactions == nil {
		transactions.Transactions = []domain.Transaction{}
	}

	return transactions, nil
}

func SaveTransactions(path string, file domain.TransactionsFile) error {
	if file.Transactions == nil {
		file.Transactions = []domain.Transaction{}
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

func AddTransaction(txType, amountInput, currency, dateValue, category, account, toAccount, note string) (domain.Transaction, error) {
	amountMinor, err := ParseAmount(amountInput)
	if err != nil {
		return domain.Transaction{}, err
	}
	if amountMinor <= 0 {
		return domain.Transaction{}, fmt.Errorf("amount must be greater than zero")
	}

	if txType != TransactionTypeExpense && txType != TransactionTypeIncome && txType != TransactionTypeTransfer {
		return domain.Transaction{}, fmt.Errorf("invalid transaction type: %s", txType)
	}

	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return domain.Transaction{}, fmt.Errorf("currency is required")
	}

	dateValue = strings.TrimSpace(dateValue)
	if dateValue == "" {
		dateValue = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", dateValue); err != nil {
		return domain.Transaction{}, fmt.Errorf("invalid date format: %s", dateValue)
	}

	category = strings.TrimSpace(category)
	account = strings.TrimSpace(account)
	toAccount = strings.TrimSpace(toAccount)
	if txType == TransactionTypeTransfer {
		if category == "" {
			category = "Transfer"
		}
	} else {
		if account == "" {
			return domain.Transaction{}, fmt.Errorf("account is required")
		}
	}

	path, err := FilePath(TransactionsFileName)
	if err != nil {
		return domain.Transaction{}, err
	}

	file, err := LoadTransactions(path)
	if err != nil {
		return domain.Transaction{}, err
	}

	localNow := time.Now()
	utcNow := localNow.UTC()
	tx := domain.Transaction{
		ID:          NewTransactionID(utcNow),
		Type:        txType,
		AmountMinor: amountMinor,
		Currency:    currency,
		Date:        dateValue,
		Category:    category,
		Account:     account,
		ToAccount:   toAccount,
		Note:        strings.TrimSpace(note),
		CreatedAt:   utcNow.Format(time.RFC3339),
		UpdatedAt:   utcNow.Format(time.RFC3339),
		Source:      "manual",
	}

	if txType == TransactionTypeTransfer {
		if err := TransferBetweenAccounts(account, toAccount, currency, amountMinor); err != nil {
			return domain.Transaction{}, err
		}
	} else {
		delta := amountMinor
		if txType == TransactionTypeExpense {
			delta = -amountMinor
		}

		if err := ApplyTransactionToAccount(account, currency, delta); err != nil {
			return domain.Transaction{}, err
		}
	}

	file.Transactions = append(file.Transactions, tx)
	if err := SaveTransactions(path, file); err != nil {
		if txType == TransactionTypeTransfer {
			_ = TransferBetweenAccounts(toAccount, account, currency, amountMinor)
		} else {
			delta := amountMinor
			if txType == TransactionTypeExpense {
				delta = -amountMinor
			}
			_ = ApplyTransactionToAccount(account, currency, -delta)
		}
		return domain.Transaction{}, err
	}

	return tx, nil
}

func EditTransaction(id string, edits TransactionEdits) (domain.Transaction, error) {
	return editTransactionWithNow(id, edits, time.Now())
}

func DeleteTransaction(id string) (domain.Transaction, error) {
	return deleteTransactionWithNow(id, time.Now())
}

func editTransactionWithNow(id string, edits TransactionEdits, now time.Time) (domain.Transaction, error) {
	txID := strings.TrimSpace(id)
	if txID == "" {
		return domain.Transaction{}, fmt.Errorf("transaction id is required")
	}
	if edits.empty() {
		return domain.Transaction{}, fmt.Errorf("no changes provided")
	}

	transactionsPath, err := FilePath(TransactionsFileName)
	if err != nil {
		return domain.Transaction{}, err
	}
	accountsPath, err := FilePath(AccountsFileName)
	if err != nil {
		return domain.Transaction{}, err
	}

	transactionsFile, err := LoadTransactions(transactionsPath)
	if err != nil {
		return domain.Transaction{}, err
	}
	accountsFile, err := LoadAccountsFile(accountsPath)
	if err != nil {
		return domain.Transaction{}, err
	}

	index := -1
	for i := range transactionsFile.Transactions {
		if transactionsFile.Transactions[i].ID == txID {
			index = i
			break
		}
	}
	if index == -1 {
		return domain.Transaction{}, fmt.Errorf("transaction '%s' not found", txID)
	}

	original := transactionsFile.Transactions[index]
	updated, changed, err := applyTransactionEdits(original, edits, now)
	if err != nil {
		return domain.Transaction{}, err
	}
	if !changed {
		return original, nil
	}

	originalAccounts := cloneAccountsFile(accountsFile)
	nowUTC := now.UTC().Format(time.RFC3339)

	if err := applyTransactionEffectToAccounts(&accountsFile, original, nowUTC, true); err != nil {
		return domain.Transaction{}, err
	}
	if err := applyTransactionEffectToAccounts(&accountsFile, updated, nowUTC, false); err != nil {
		if rollbackErr := applyTransactionEffectToAccounts(&accountsFile, original, nowUTC, false); rollbackErr != nil {
			return domain.Transaction{}, fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
		}
		return domain.Transaction{}, err
	}

	transactionsFile.Transactions[index] = updated
	if err := SaveAccountsFile(accountsPath, accountsFile); err != nil {
		return domain.Transaction{}, err
	}
	if err := SaveTransactions(transactionsPath, transactionsFile); err != nil {
		if rollbackErr := SaveAccountsFile(accountsPath, originalAccounts); rollbackErr != nil {
			return domain.Transaction{}, fmt.Errorf("save transactions: %w; rollback accounts: %v", err, rollbackErr)
		}
		return domain.Transaction{}, err
	}

	return updated, nil
}

func deleteTransactionWithNow(id string, now time.Time) (domain.Transaction, error) {
	txID := strings.TrimSpace(id)
	if txID == "" {
		return domain.Transaction{}, fmt.Errorf("transaction id is required")
	}

	transactionsPath, err := FilePath(TransactionsFileName)
	if err != nil {
		return domain.Transaction{}, err
	}
	accountsPath, err := FilePath(AccountsFileName)
	if err != nil {
		return domain.Transaction{}, err
	}

	transactionsFile, err := LoadTransactions(transactionsPath)
	if err != nil {
		return domain.Transaction{}, err
	}
	accountsFile, err := LoadAccountsFile(accountsPath)
	if err != nil {
		return domain.Transaction{}, err
	}

	index := -1
	for i := range transactionsFile.Transactions {
		if transactionsFile.Transactions[i].ID == txID {
			index = i
			break
		}
	}
	if index == -1 {
		return domain.Transaction{}, fmt.Errorf("transaction '%s' not found", txID)
	}

	deleted := transactionsFile.Transactions[index]
	originalAccounts := cloneAccountsFile(accountsFile)
	nowUTC := now.UTC().Format(time.RFC3339)

	if err := applyTransactionEffectToAccounts(&accountsFile, deleted, nowUTC, true); err != nil {
		return domain.Transaction{}, err
	}

	transactionsFile.Transactions = append(transactionsFile.Transactions[:index], transactionsFile.Transactions[index+1:]...)
	if err := SaveAccountsFile(accountsPath, accountsFile); err != nil {
		return domain.Transaction{}, err
	}
	if err := SaveTransactions(transactionsPath, transactionsFile); err != nil {
		if rollbackErr := SaveAccountsFile(accountsPath, originalAccounts); rollbackErr != nil {
			return domain.Transaction{}, fmt.Errorf("save transactions: %w; rollback accounts: %v", err, rollbackErr)
		}
		return domain.Transaction{}, err
	}

	return deleted, nil
}

func applyTransactionEdits(tx domain.Transaction, edits TransactionEdits, now time.Time) (domain.Transaction, bool, error) {
	updated := tx

	if edits.Date != nil {
		updated.Date = *edits.Date
	}
	if edits.Amount != nil {
		amountMinor, err := ParseAmount(*edits.Amount)
		if err != nil {
			return domain.Transaction{}, false, err
		}
		if amountMinor <= 0 {
			return domain.Transaction{}, false, fmt.Errorf("amount must be greater than zero")
		}
		updated.AmountMinor = amountMinor
	}
	if edits.Category != nil {
		updated.Category = *edits.Category
	}
	if edits.Account != nil {
		updated.Account = *edits.Account
	}
	if edits.ToAccount != nil {
		updated.ToAccount = *edits.ToAccount
	}
	if edits.Note != nil {
		updated.Note = *edits.Note
	}

	updated.Type = strings.TrimSpace(updated.Type)
	if updated.Type != TransactionTypeExpense && updated.Type != TransactionTypeIncome && updated.Type != TransactionTypeTransfer {
		return domain.Transaction{}, false, fmt.Errorf("invalid transaction type: %s", updated.Type)
	}

	updated.Currency = strings.ToUpper(strings.TrimSpace(updated.Currency))
	if updated.Currency == "" {
		return domain.Transaction{}, false, fmt.Errorf("currency is required")
	}

	updated.Date = strings.TrimSpace(updated.Date)
	if updated.Date == "" {
		return domain.Transaction{}, false, fmt.Errorf("date is required")
	}
	if _, err := time.Parse("2006-01-02", updated.Date); err != nil {
		return domain.Transaction{}, false, fmt.Errorf("invalid date format: %s", updated.Date)
	}

	updated.Category = strings.TrimSpace(updated.Category)
	updated.Account = strings.TrimSpace(updated.Account)
	updated.ToAccount = strings.TrimSpace(updated.ToAccount)
	updated.Note = strings.TrimSpace(updated.Note)

	if updated.Type == TransactionTypeTransfer {
		if updated.Category == "" {
			updated.Category = "Transfer"
		}
		if updated.Account == "" || updated.ToAccount == "" {
			return domain.Transaction{}, false, fmt.Errorf("both source and destination accounts are required")
		}
		if strings.EqualFold(updated.Account, updated.ToAccount) {
			return domain.Transaction{}, false, fmt.Errorf("source and destination accounts must be different")
		}
	} else {
		if updated.Account == "" {
			return domain.Transaction{}, false, fmt.Errorf("account is required")
		}
		if edits.ToAccount != nil && strings.TrimSpace(*edits.ToAccount) != "" {
			return domain.Transaction{}, false, fmt.Errorf("to-account can only be edited for transfer transactions")
		}
		updated.ToAccount = ""
	}

	changed := updated.Date != tx.Date ||
		updated.AmountMinor != tx.AmountMinor ||
		updated.Category != tx.Category ||
		updated.Account != tx.Account ||
		updated.ToAccount != tx.ToAccount ||
		updated.Note != tx.Note
	if !changed {
		return tx, false, nil
	}

	updated.UpdatedAt = now.UTC().Format(time.RFC3339)
	return updated, true, nil
}

func applyTransactionEffectToAccounts(accountsFile *domain.AccountsFile, tx domain.Transaction, now string, reverse bool) error {
	switch tx.Type {
	case TransactionTypeTransfer:
		from := tx.Account
		to := tx.ToAccount
		if reverse {
			from, to = to, from
		}
		return transferBetweenAccountsInFile(accountsFile, from, to, tx.Currency, tx.AmountMinor, now)
	case TransactionTypeExpense, TransactionTypeIncome:
		delta := tx.AmountMinor
		if tx.Type == TransactionTypeExpense {
			delta = -delta
		}
		if reverse {
			delta = -delta
		}
		return applyAccountDeltaToFile(accountsFile, tx.Account, tx.Currency, delta, now)
	default:
		return fmt.Errorf("invalid transaction type: %s", tx.Type)
	}
}

func cloneAccountsFile(file domain.AccountsFile) domain.AccountsFile {
	cloned := domain.AccountsFile{
		SchemaVersion: file.SchemaVersion,
		Accounts:      make([]domain.Account, len(file.Accounts)),
	}
	copy(cloned.Accounts, file.Accounts)
	return cloned
}

func (e TransactionEdits) empty() bool {
	return e.Date == nil && e.Amount == nil && e.Category == nil && e.Account == nil && e.ToAccount == nil && e.Note == nil
}

func FormatMinor(amountMinor int64) string {
	negative := amountMinor < 0
	if negative {
		amountMinor = -amountMinor
	}

	whole := amountMinor / 100
	fraction := amountMinor % 100
	wholeFormatted := formatWithCommas(whole)

	if negative {
		return fmt.Sprintf("-%s.%02d", wholeFormatted, fraction)
	}

	return fmt.Sprintf("%s.%02d", wholeFormatted, fraction)
}

func formatWithCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}

	first := len(s) % 3
	if first == 0 {
		first = 3
	}

	result := s[:first]
	for i := first; i < len(s); i += 3 {
		result += "," + s[i:i+3]
	}

	return result
}
