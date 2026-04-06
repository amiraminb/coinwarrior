package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
)

func ParseAmount(input string) (int64, error) {
	amount := strings.TrimSpace(input)
	if amount == "" {
		return 0, fmt.Errorf("amount cannot be empty")
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

	return wholeValue*100 + fracValue, nil
}

func NewTransactionID(now time.Time) string {
	return fmt.Sprintf("txn_%d", now.UnixNano())
}

func LoadTransactions(path string) (model.TransactionsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.TransactionsFile{SchemaVersion: 1, Transactions: []model.Transaction{}}, nil
		}
		return model.TransactionsFile{}, err
	}

	var transactions model.TransactionsFile
	if err := json.Unmarshal(data, &transactions); err != nil {
		return model.TransactionsFile{}, err
	}
	if transactions.Transactions == nil {
		transactions.Transactions = []model.Transaction{}
	}

	return transactions, nil
}

func SaveTransactions(path string, file model.TransactionsFile) error {
	if file.Transactions == nil {
		file.Transactions = []model.Transaction{}
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

func AddTransaction(txType, amountInput, currency, category, account string) (model.Transaction, error) {
	amountMinor, err := ParseAmount(amountInput)
	if err != nil {
		return model.Transaction{}, err
	}

	path, err := FilePath(TransactionsFileName)
	if err != nil {
		return model.Transaction{}, err
	}

	file, err := LoadTransactions(path)
	if err != nil {
		return model.Transaction{}, err
	}

	localNow := time.Now()
	utcNow := localNow.UTC()
	tx := model.Transaction{
		ID:          NewTransactionID(utcNow),
		Type:        txType,
		AmountMinor: amountMinor,
		Currency:    strings.ToUpper(currency),
		Date:        localNow.Format("2006-01-02"),
		Category:    strings.TrimSpace(category),
		Account:     strings.TrimSpace(account),
		CreatedAt:   utcNow.Format(time.RFC3339),
		UpdatedAt:   utcNow.Format(time.RFC3339),
		Source:      "manual",
	}

	file.Transactions = append(file.Transactions, tx)
	if err := SaveTransactions(path, file); err != nil {
		return model.Transaction{}, err
	}

	return tx, nil
}

func FormatMinor(amountMinor int64) string {
	negative := amountMinor < 0
	if negative {
		amountMinor = -amountMinor
	}

	whole := amountMinor / 100
	fraction := amountMinor % 100

	if negative {
		return fmt.Sprintf("-%d.%02d", whole, fraction)
	}

	return fmt.Sprintf("%d.%02d", whole, fraction)
}
