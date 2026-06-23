package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
)

func (s *Service) LoadAccounts() ([]model.Account, error) {
	return s.repo.LoadAccounts()
}

func (s *Service) LoadAccountNames() ([]string, error) {
	accounts, err := s.repo.LoadAccounts()
	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	for _, account := range accounts {
		result = append(result, account.Name)
	}

	return result, nil
}

func (s *Service) AddAccount(name, currency, openingBalanceInput string) (model.Account, error) {
	accountName := strings.TrimSpace(name)
	cur := money.NormalizeCurrency(currency)
	if accountName == "" {
		return model.Account{}, fmt.Errorf("account name is required")
	}
	if cur == "" {
		return model.Account{}, fmt.Errorf("currency is required")
	}

	balanceMinor, err := money.Parse(openingBalanceInput)
	if err != nil {
		return model.Account{}, err
	}

	accounts, err := s.repo.LoadAccounts()
	if err != nil {
		return model.Account{}, err
	}

	for _, existing := range accounts {
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

	accounts = append(accounts, account)
	if err := s.repo.SaveAccounts(accounts); err != nil {
		return model.Account{}, err
	}

	return account, nil
}

func (s *Service) UpdateAccountBalance(name, amountInput string) (model.Account, error) {
	accountName := strings.TrimSpace(name)
	if accountName == "" {
		return model.Account{}, fmt.Errorf("account name is required")
	}

	balanceMinor, err := money.Parse(amountInput)
	if err != nil {
		return model.Account{}, err
	}

	accounts, err := s.repo.LoadAccounts()
	if err != nil {
		return model.Account{}, err
	}

	for i := range accounts {
		if strings.EqualFold(accounts[i].Name, accountName) {
			accounts[i].BalanceMinor = balanceMinor
			accounts[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := s.repo.SaveAccounts(accounts); err != nil {
				return model.Account{}, err
			}
			return accounts[i], nil
		}
	}

	return model.Account{}, fmt.Errorf("account '%s' not found", accountName)
}

func applyAccountDeltaToFile(accounts []model.Account, accountName, currency string, deltaMinor int64, now string) error {
	name := strings.TrimSpace(accountName)
	cur := money.NormalizeCurrency(currency)
	if name == "" {
		return fmt.Errorf("account is required")
	}
	if cur == "" {
		return fmt.Errorf("currency is required")
	}

	for i := range accounts {
		if strings.EqualFold(accounts[i].Name, name) {
			if !strings.EqualFold(accounts[i].Currency, cur) {
				return fmt.Errorf("account '%s' uses currency %s, got %s", accounts[i].Name, accounts[i].Currency, cur)
			}
			accounts[i].BalanceMinor += deltaMinor
			accounts[i].UpdatedAt = now
			return nil
		}
	}

	return fmt.Errorf("account '%s' not found", name)
}

func transferBetweenAccountsInFile(accounts []model.Account, fromAccount, toAccount, currency string, amountMinor int64, now string) error {
	from := strings.TrimSpace(fromAccount)
	to := strings.TrimSpace(toAccount)
	cur := money.NormalizeCurrency(currency)

	if from == "" || to == "" {
		return fmt.Errorf("both source and destination accounts are required")
	}
	if strings.EqualFold(from, to) {
		return fmt.Errorf("source and destination accounts must be different")
	}
	if amountMinor <= 0 {
		return fmt.Errorf("transfer amount must be greater than zero")
	}

	fromIdx := -1
	toIdx := -1
	for i := range accounts {
		if strings.EqualFold(accounts[i].Name, from) {
			fromIdx = i
		}
		if strings.EqualFold(accounts[i].Name, to) {
			toIdx = i
		}
	}

	if fromIdx == -1 {
		return fmt.Errorf("account '%s' not found", from)
	}
	if toIdx == -1 {
		return fmt.Errorf("account '%s' not found", to)
	}

	fromCurrency := money.NormalizeCurrency(accounts[fromIdx].Currency)
	toCurrency := money.NormalizeCurrency(accounts[toIdx].Currency)
	if fromCurrency != cur {
		return fmt.Errorf("account '%s' uses currency %s, got %s", accounts[fromIdx].Name, fromCurrency, cur)
	}
	if toCurrency != cur {
		return fmt.Errorf("account '%s' uses currency %s, got %s", accounts[toIdx].Name, toCurrency, cur)
	}

	accounts[fromIdx].BalanceMinor -= amountMinor
	accounts[toIdx].BalanceMinor += amountMinor
	accounts[fromIdx].UpdatedAt = now
	accounts[toIdx].UpdatedAt = now

	return nil
}
