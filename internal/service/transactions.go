package service

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
)

func findTransactionIndex(transactions []model.Transaction, id string) int {
	return slices.IndexFunc(transactions, func(tx model.Transaction) bool {
		return tx.ID == id
	})
}

type ledgerMutator func(transactions *[]model.Transaction, accounts *[]model.Account, nowUTC string) error

func (s *Service) mutateLedger(now time.Time, mutate ledgerMutator) error {
	transactions, err := s.repo.LoadTransactions()
	if err != nil {
		return err
	}
	accounts, err := s.repo.LoadAccounts()
	if err != nil {
		return err
	}

	originalAccounts := cloneAccounts(accounts)
	nowUTC := now.UTC().Format(time.RFC3339)

	if err := mutate(&transactions, &accounts, nowUTC); err != nil {
		return err
	}

	if err := s.repo.SaveAccounts(accounts); err != nil {
		return err
	}
	if err := s.repo.SaveTransactions(transactions); err != nil {
		if rollbackErr := s.repo.SaveAccounts(originalAccounts); rollbackErr != nil {
			return fmt.Errorf("save transactions: %w; rollback accounts: %v", err, rollbackErr)
		}
		return err
	}

	return nil
}

type TransactionEdits struct {
	Date      *string
	Amount    *string
	Category  *string
	Account   *string
	ToAccount *string
	Note      *string
}

func NewTransactionID(now time.Time) string {
	return fmt.Sprintf("txn_%d", now.UnixNano())
}

func (s *Service) AddTransaction(txType, amountInput, currency, dateValue, category, account, toAccount, note string) (model.Transaction, error) {
	amountMinor, err := money.Parse(amountInput)
	if err != nil {
		return model.Transaction{}, err
	}
	if amountMinor <= 0 {
		return model.Transaction{}, fmt.Errorf("amount must be greater than zero")
	}

	if txType != model.TransactionTypeExpense && txType != model.TransactionTypeIncome && txType != model.TransactionTypeTransfer {
		return model.Transaction{}, fmt.Errorf("invalid transaction type: %s", txType)
	}

	currency = money.NormalizeCurrency(currency)
	if currency == "" {
		return model.Transaction{}, fmt.Errorf("currency is required")
	}

	dateValue = strings.TrimSpace(dateValue)
	if dateValue == "" {
		dateValue = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", dateValue); err != nil {
		return model.Transaction{}, fmt.Errorf("invalid date format: %s", dateValue)
	}

	category = strings.TrimSpace(category)
	account = strings.TrimSpace(account)
	toAccount = strings.TrimSpace(toAccount)
	if txType == model.TransactionTypeTransfer {
		if category == "" {
			category = model.TransferCategory
		}
	} else {
		if account == "" {
			return model.Transaction{}, fmt.Errorf("account is required")
		}
	}

	now := time.Now()
	utcNow := now.UTC()
	tx := model.Transaction{
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
		Source:      model.TransactionSourceManual,
	}

	err = s.mutateLedger(now, func(transactions *[]model.Transaction, accounts *[]model.Account, nowUTC string) error {
		if err := applyTransactionEffect(*accounts, tx, nowUTC); err != nil {
			return err
		}
		*transactions = append(*transactions, tx)
		return nil
	})
	if err != nil {
		return model.Transaction{}, err
	}
	return tx, nil
}

func (s *Service) EditTransaction(id string, edits TransactionEdits) (model.Transaction, error) {
	return s.editTransactionWithNow(id, edits, time.Now())
}

func (s *Service) DeleteTransaction(id string) (model.Transaction, error) {
	return s.deleteTransactionWithNow(id, time.Now())
}

func (s *Service) editTransactionWithNow(id string, edits TransactionEdits, now time.Time) (model.Transaction, error) {
	txID := strings.TrimSpace(id)
	if txID == "" {
		return model.Transaction{}, fmt.Errorf("transaction id is required")
	}
	if edits.empty() {
		return model.Transaction{}, fmt.Errorf("no changes provided")
	}

	var result model.Transaction
	err := s.mutateLedger(now, func(transactions *[]model.Transaction, accounts *[]model.Account, nowUTC string) error {
		index := findTransactionIndex(*transactions, txID)
		if index == -1 {
			return fmt.Errorf("transaction '%s' not found", txID)
		}

		original := (*transactions)[index]
		updated, changed, err := applyTransactionEdits(original, edits, now)
		if err != nil {
			return err
		}
		if !changed {
			result = original
			return nil
		}

		if err := revertTransactionEffect(*accounts, original, nowUTC); err != nil {
			return err
		}
		if err := applyTransactionEffect(*accounts, updated, nowUTC); err != nil {
			return err
		}

		(*transactions)[index] = updated
		result = updated
		return nil
	})
	if err != nil {
		return model.Transaction{}, err
	}
	return result, nil
}

func (s *Service) deleteTransactionWithNow(id string, now time.Time) (model.Transaction, error) {
	txID := strings.TrimSpace(id)
	if txID == "" {
		return model.Transaction{}, fmt.Errorf("transaction id is required")
	}

	var deleted model.Transaction
	err := s.mutateLedger(now, func(transactions *[]model.Transaction, accounts *[]model.Account, nowUTC string) error {
		index := findTransactionIndex(*transactions, txID)
		if index == -1 {
			return fmt.Errorf("transaction '%s' not found", txID)
		}

		deleted = (*transactions)[index]
		if err := revertTransactionEffect(*accounts, deleted, nowUTC); err != nil {
			return err
		}
		*transactions = append((*transactions)[:index], (*transactions)[index+1:]...)
		return nil
	})
	if err != nil {
		return model.Transaction{}, err
	}
	return deleted, nil
}

func applyTransactionEdits(tx model.Transaction, edits TransactionEdits, now time.Time) (model.Transaction, bool, error) {
	updated := tx

	if edits.Date != nil {
		updated.Date = *edits.Date
	}
	if edits.Amount != nil {
		amountMinor, err := money.Parse(*edits.Amount)
		if err != nil {
			return model.Transaction{}, false, err
		}
		if amountMinor <= 0 {
			return model.Transaction{}, false, fmt.Errorf("amount must be greater than zero")
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
	if updated.Type != model.TransactionTypeExpense && updated.Type != model.TransactionTypeIncome && updated.Type != model.TransactionTypeTransfer {
		return model.Transaction{}, false, fmt.Errorf("invalid transaction type: %s", updated.Type)
	}

	updated.Currency = money.NormalizeCurrency(updated.Currency)
	if updated.Currency == "" {
		return model.Transaction{}, false, fmt.Errorf("currency is required")
	}

	updated.Date = strings.TrimSpace(updated.Date)
	if updated.Date == "" {
		return model.Transaction{}, false, fmt.Errorf("date is required")
	}
	if _, err := time.Parse("2006-01-02", updated.Date); err != nil {
		return model.Transaction{}, false, fmt.Errorf("invalid date format: %s", updated.Date)
	}

	updated.Category = strings.TrimSpace(updated.Category)
	updated.Account = strings.TrimSpace(updated.Account)
	updated.ToAccount = strings.TrimSpace(updated.ToAccount)
	updated.Note = strings.TrimSpace(updated.Note)

	if updated.Type == model.TransactionTypeTransfer {
		if updated.Category == "" {
			updated.Category = model.TransferCategory
		}
		if updated.Account == "" || updated.ToAccount == "" {
			return model.Transaction{}, false, fmt.Errorf("both source and destination accounts are required")
		}
		if strings.EqualFold(updated.Account, updated.ToAccount) {
			return model.Transaction{}, false, fmt.Errorf("source and destination accounts must be different")
		}
	} else {
		if updated.Account == "" {
			return model.Transaction{}, false, fmt.Errorf("account is required")
		}
		if edits.ToAccount != nil && strings.TrimSpace(*edits.ToAccount) != "" {
			return model.Transaction{}, false, fmt.Errorf("to-account can only be edited for transfer transactions")
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

func applyTransactionEffect(accounts []model.Account, tx model.Transaction, now string) error {
	switch tx.Type {
	case model.TransactionTypeTransfer:
		return transferBetweenAccountsInFile(accounts, tx.Account, tx.ToAccount, tx.Currency, tx.AmountMinor, now)
	case model.TransactionTypeExpense, model.TransactionTypeIncome:
		delta := tx.AmountMinor
		if tx.Type == model.TransactionTypeExpense {
			delta = -delta
		}
		return applyAccountDeltaToFile(accounts, tx.Account, tx.Currency, delta, now)
	default:
		return fmt.Errorf("invalid transaction type: %s", tx.Type)
	}
}

func revertTransactionEffect(accounts []model.Account, tx model.Transaction, now string) error {
	switch tx.Type {
	case model.TransactionTypeTransfer:
		return transferBetweenAccountsInFile(accounts, tx.ToAccount, tx.Account, tx.Currency, tx.AmountMinor, now)
	case model.TransactionTypeExpense, model.TransactionTypeIncome:
		delta := tx.AmountMinor
		if tx.Type == model.TransactionTypeIncome {
			delta = -delta
		}
		return applyAccountDeltaToFile(accounts, tx.Account, tx.Currency, delta, now)
	default:
		return fmt.Errorf("invalid transaction type: %s", tx.Type)
	}
}

func cloneAccounts(accounts []model.Account) []model.Account {
	cloned := make([]model.Account, len(accounts))
	copy(cloned, accounts)
	return cloned
}

func (e TransactionEdits) empty() bool {
	return e.Date == nil && e.Amount == nil && e.Category == nil && e.Account == nil && e.ToAccount == nil && e.Note == nil
}

