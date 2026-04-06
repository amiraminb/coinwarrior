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

func AddTransaction(txType, amountInput, currency, dateValue, category, account, toAccount, note string) (model.Transaction, error) {
	amountMinor, err := ParseAmount(amountInput)
	if err != nil {
		return model.Transaction{}, err
	}
	if amountMinor <= 0 {
		return model.Transaction{}, fmt.Errorf("amount must be greater than zero")
	}

	if txType != TransactionTypeExpense && txType != TransactionTypeIncome && txType != TransactionTypeTransfer {
		return model.Transaction{}, fmt.Errorf("invalid transaction type: %s", txType)
	}

	currency = strings.ToUpper(strings.TrimSpace(currency))
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
	if txType == TransactionTypeTransfer {
		if category == "" {
			category = "Transfer"
		}
	} else {
		if account == "" {
			return model.Transaction{}, fmt.Errorf("account is required")
		}
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
			return model.Transaction{}, err
		}
	} else {
		delta := amountMinor
		if txType == TransactionTypeExpense {
			delta = -amountMinor
		}

		if err := ApplyTransactionToAccount(account, currency, delta); err != nil {
			return model.Transaction{}, err
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
