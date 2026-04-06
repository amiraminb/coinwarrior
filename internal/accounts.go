package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
)

func LoadAccounts() ([]string, error) {
	path, err := FilePath(AccountsFileName)
	if err != nil {
		return nil, err
	}

	accountsFile, err := LoadAccountsFile(path)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	for _, account := range accountsFile.Accounts {
		result = append(result, account.Name)
	}

	return result, nil
}

func AccountExists(accounts []string, account string) bool {
	for _, existing := range accounts {
		if strings.EqualFold(existing, account) {
			return true
		}
	}
	return false
}

func LoadAccountsFile(path string) (model.AccountsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.AccountsFile{SchemaVersion: 1, Accounts: []model.Account{}}, nil
		}
		return model.AccountsFile{}, err
	}

	var accounts model.AccountsFile
	if err := json.Unmarshal(data, &accounts); err != nil {
		return model.AccountsFile{}, err
	}
	if accounts.Accounts == nil {
		accounts.Accounts = []model.Account{}
	}

	return accounts, nil
}

func SaveAccountsFile(path string, accounts model.AccountsFile) error {
	if accounts.Accounts == nil {
		accounts.Accounts = []model.Account{}
	}

	data, err := json.MarshalIndent(accounts, "", "  ")
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

func ApplyTransactionToAccount(accountName, currency string, deltaMinor int64) error {
	name := strings.TrimSpace(accountName)
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if name == "" {
		return fmt.Errorf("account is required")
	}
	if cur == "" {
		return fmt.Errorf("currency is required")
	}

	path, err := FilePath(AccountsFileName)
	if err != nil {
		return err
	}

	accountsFile, err := LoadAccountsFile(path)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)

	for i := range accountsFile.Accounts {
		if strings.EqualFold(accountsFile.Accounts[i].Name, name) {
			if !strings.EqualFold(accountsFile.Accounts[i].Currency, cur) {
				return fmt.Errorf("account '%s' uses currency %s, got %s", accountsFile.Accounts[i].Name, accountsFile.Accounts[i].Currency, cur)
			}
			accountsFile.Accounts[i].BalanceMinor += deltaMinor
			accountsFile.Accounts[i].UpdatedAt = now
			return SaveAccountsFile(path, accountsFile)
		}
	}

	return fmt.Errorf("account '%s' not found", name)
}

func TransferBetweenAccounts(fromAccount, toAccount, currency string, amountMinor int64) error {
	from := strings.TrimSpace(fromAccount)
	to := strings.TrimSpace(toAccount)
	cur := strings.ToUpper(strings.TrimSpace(currency))

	if from == "" || to == "" {
		return fmt.Errorf("both source and destination accounts are required")
	}
	if strings.EqualFold(from, to) {
		return fmt.Errorf("source and destination accounts must be different")
	}
	if amountMinor <= 0 {
		return fmt.Errorf("transfer amount must be greater than zero")
	}

	path, err := FilePath(AccountsFileName)
	if err != nil {
		return err
	}

	accountsFile, err := LoadAccountsFile(path)
	if err != nil {
		return err
	}

	fromIdx := -1
	toIdx := -1
	for i := range accountsFile.Accounts {
		if strings.EqualFold(accountsFile.Accounts[i].Name, from) {
			fromIdx = i
		}
		if strings.EqualFold(accountsFile.Accounts[i].Name, to) {
			toIdx = i
		}
	}

	if fromIdx == -1 {
		return fmt.Errorf("account '%s' not found", from)
	}
	if toIdx == -1 {
		return fmt.Errorf("account '%s' not found", to)
	}

	fromCurrency := strings.ToUpper(strings.TrimSpace(accountsFile.Accounts[fromIdx].Currency))
	toCurrency := strings.ToUpper(strings.TrimSpace(accountsFile.Accounts[toIdx].Currency))
	if fromCurrency != cur {
		return fmt.Errorf("account '%s' uses currency %s, got %s", accountsFile.Accounts[fromIdx].Name, fromCurrency, cur)
	}
	if toCurrency != cur {
		return fmt.Errorf("account '%s' uses currency %s, got %s", accountsFile.Accounts[toIdx].Name, toCurrency, cur)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	accountsFile.Accounts[fromIdx].BalanceMinor -= amountMinor
	accountsFile.Accounts[toIdx].BalanceMinor += amountMinor
	accountsFile.Accounts[fromIdx].UpdatedAt = now
	accountsFile.Accounts[toIdx].UpdatedAt = now

	return SaveAccountsFile(path, accountsFile)
}

func AddAccount(name, currency, openingBalanceInput string) (model.Account, error) {
	accountName := strings.TrimSpace(name)
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if accountName == "" {
		return model.Account{}, fmt.Errorf("account name is required")
	}
	if cur == "" {
		return model.Account{}, fmt.Errorf("currency is required")
	}

	balanceMinor, err := ParseAmount(openingBalanceInput)
	if err != nil {
		return model.Account{}, err
	}

	path, err := FilePath(AccountsFileName)
	if err != nil {
		return model.Account{}, err
	}

	accountsFile, err := LoadAccountsFile(path)
	if err != nil {
		return model.Account{}, err
	}

	for _, existing := range accountsFile.Accounts {
		if strings.EqualFold(existing.Name, accountName) {
			return model.Account{}, fmt.Errorf("account '%s' already exists", existing.Name)
		}
	}

	account := model.Account{
		Name:         accountName,
		Currency:     cur,
		BalanceMinor: balanceMinor,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	accountsFile.Accounts = append(accountsFile.Accounts, account)
	if err := SaveAccountsFile(path, accountsFile); err != nil {
		return model.Account{}, err
	}

	return account, nil
}

func UpdateAccountBalance(name, amountInput string) (model.Account, error) {
	accountName := strings.TrimSpace(name)
	if accountName == "" {
		return model.Account{}, fmt.Errorf("account name is required")
	}

	balanceMinor, err := ParseAmount(amountInput)
	if err != nil {
		return model.Account{}, err
	}

	path, err := FilePath(AccountsFileName)
	if err != nil {
		return model.Account{}, err
	}

	accountsFile, err := LoadAccountsFile(path)
	if err != nil {
		return model.Account{}, err
	}

	for i := range accountsFile.Accounts {
		if strings.EqualFold(accountsFile.Accounts[i].Name, accountName) {
			accountsFile.Accounts[i].BalanceMinor = balanceMinor
			accountsFile.Accounts[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := SaveAccountsFile(path, accountsFile); err != nil {
				return model.Account{}, err
			}
			return accountsFile.Accounts[i], nil
		}
	}

	return model.Account{}, fmt.Errorf("account '%s' not found", accountName)
}
