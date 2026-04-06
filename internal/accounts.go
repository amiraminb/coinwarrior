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

	accountsFile.Accounts = append(accountsFile.Accounts, model.Account{
		Name:         name,
		Currency:     cur,
		BalanceMinor: deltaMinor,
		UpdatedAt:    now,
	})

	return SaveAccountsFile(path, accountsFile)
}
