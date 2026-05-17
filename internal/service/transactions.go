package service

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/domain"
	"github.com/amiraminb/coinwarrior/internal/money"
)

func findTransactionIndex(transactions []domain.Transaction, id string) int {
	return slices.IndexFunc(transactions, func(tx domain.Transaction) bool {
		return tx.ID == id
	})
}

type ledgerMutator func(transactions *[]domain.Transaction, accounts *[]domain.Account, nowUTC string) error

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

func (s *Service) AddTransaction(txType, amountInput, currency, dateValue, category, account, toAccount, note string) (domain.Transaction, error) {
	amountMinor, err := money.Parse(amountInput)
	if err != nil {
		return domain.Transaction{}, err
	}
	if amountMinor <= 0 {
		return domain.Transaction{}, fmt.Errorf("amount must be greater than zero")
	}

	if txType != domain.TransactionTypeExpense && txType != domain.TransactionTypeIncome && txType != domain.TransactionTypeTransfer {
		return domain.Transaction{}, fmt.Errorf("invalid transaction type: %s", txType)
	}

	currency = money.NormalizeCurrency(currency)
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
	if txType == domain.TransactionTypeTransfer {
		if category == "" {
			category = domain.TransferCategory
		}
	} else {
		if account == "" {
			return domain.Transaction{}, fmt.Errorf("account is required")
		}
	}

	now := time.Now()
	utcNow := now.UTC()
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
		Source:      domain.TransactionSourceManual,
	}

	err = s.mutateLedger(now, func(transactions *[]domain.Transaction, accounts *[]domain.Account, nowUTC string) error {
		if err := applyTransactionEffect(*accounts, tx, nowUTC); err != nil {
			return err
		}
		*transactions = append(*transactions, tx)
		return nil
	})
	if err != nil {
		return domain.Transaction{}, err
	}
	return tx, nil
}

func (s *Service) EditTransaction(id string, edits TransactionEdits) (domain.Transaction, error) {
	return s.editTransactionWithNow(id, edits, time.Now())
}

func (s *Service) DeleteTransaction(id string) (domain.Transaction, error) {
	return s.deleteTransactionWithNow(id, time.Now())
}

func (s *Service) editTransactionWithNow(id string, edits TransactionEdits, now time.Time) (domain.Transaction, error) {
	txID := strings.TrimSpace(id)
	if txID == "" {
		return domain.Transaction{}, fmt.Errorf("transaction id is required")
	}
	if edits.empty() {
		return domain.Transaction{}, fmt.Errorf("no changes provided")
	}

	var result domain.Transaction
	err := s.mutateLedger(now, func(transactions *[]domain.Transaction, accounts *[]domain.Account, nowUTC string) error {
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
		return domain.Transaction{}, err
	}
	return result, nil
}

func (s *Service) deleteTransactionWithNow(id string, now time.Time) (domain.Transaction, error) {
	txID := strings.TrimSpace(id)
	if txID == "" {
		return domain.Transaction{}, fmt.Errorf("transaction id is required")
	}

	var deleted domain.Transaction
	err := s.mutateLedger(now, func(transactions *[]domain.Transaction, accounts *[]domain.Account, nowUTC string) error {
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
		amountMinor, err := money.Parse(*edits.Amount)
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
	if updated.Type != domain.TransactionTypeExpense && updated.Type != domain.TransactionTypeIncome && updated.Type != domain.TransactionTypeTransfer {
		return domain.Transaction{}, false, fmt.Errorf("invalid transaction type: %s", updated.Type)
	}

	updated.Currency = money.NormalizeCurrency(updated.Currency)
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

	if updated.Type == domain.TransactionTypeTransfer {
		if updated.Category == "" {
			updated.Category = domain.TransferCategory
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

func applyTransactionEffect(accounts []domain.Account, tx domain.Transaction, now string) error {
	switch tx.Type {
	case domain.TransactionTypeTransfer:
		return transferBetweenAccountsInFile(accounts, tx.Account, tx.ToAccount, tx.Currency, tx.AmountMinor, now)
	case domain.TransactionTypeExpense, domain.TransactionTypeIncome:
		delta := tx.AmountMinor
		if tx.Type == domain.TransactionTypeExpense {
			delta = -delta
		}
		return applyAccountDeltaToFile(accounts, tx.Account, tx.Currency, delta, now)
	default:
		return fmt.Errorf("invalid transaction type: %s", tx.Type)
	}
}

func revertTransactionEffect(accounts []domain.Account, tx domain.Transaction, now string) error {
	switch tx.Type {
	case domain.TransactionTypeTransfer:
		return transferBetweenAccountsInFile(accounts, tx.ToAccount, tx.Account, tx.Currency, tx.AmountMinor, now)
	case domain.TransactionTypeExpense, domain.TransactionTypeIncome:
		delta := tx.AmountMinor
		if tx.Type == domain.TransactionTypeIncome {
			delta = -delta
		}
		return applyAccountDeltaToFile(accounts, tx.Account, tx.Currency, delta, now)
	default:
		return fmt.Errorf("invalid transaction type: %s", tx.Type)
	}
}

func cloneAccounts(accounts []domain.Account) []domain.Account {
	cloned := make([]domain.Account, len(accounts))
	copy(cloned, accounts)
	return cloned
}

func (e TransactionEdits) empty() bool {
	return e.Date == nil && e.Amount == nil && e.Category == nil && e.Account == nil && e.ToAccount == nil && e.Note == nil
}

