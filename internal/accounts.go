package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/domain"
	"github.com/amiraminb/coinwarrior/internal/repository"
)

func LoadAccounts() ([]string, error) {
	accounts, err := repository.FRepository.LoadAccounts()
	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	for _, account := range accounts {
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

func ApplyTransactionToAccount(accountName, currency string, deltaMinor int64) error {
	name := strings.TrimSpace(accountName)
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if name == "" {
		return fmt.Errorf("account is required")
	}
	if cur == "" {
		return fmt.Errorf("currency is required")
	}

	accounts, err := repository.FRepository.LoadAccounts()
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := applyAccountDeltaToFile(accounts, name, cur, deltaMinor, now); err != nil {
		return err
	}

	return repository.FRepository.SaveAccounts(accounts)
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

	accounts, err := repository.FRepository.LoadAccounts()
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := transferBetweenAccountsInFile(accounts, from, to, cur, amountMinor, now); err != nil {
		return err
	}

	return repository.FRepository.SaveAccounts(accounts)
}

func AddAccount(name, currency, openingBalanceInput string) (domain.Account, error) {
	accountName := strings.TrimSpace(name)
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if accountName == "" {
		return domain.Account{}, fmt.Errorf("account name is required")
	}
	if cur == "" {
		return domain.Account{}, fmt.Errorf("currency is required")
	}

	balanceMinor, err := ParseAmount(openingBalanceInput)
	if err != nil {
		return domain.Account{}, err
	}

	accounts, err := repository.FRepository.LoadAccounts()
	if err != nil {
		return domain.Account{}, err
	}

	for _, existing := range accounts {
		if strings.EqualFold(existing.Name, accountName) {
			return domain.Account{}, fmt.Errorf("account '%s' already exists", existing.Name)
		}
	}

	account := domain.Account{
		Name:         accountName,
		Currency:     cur,
		BalanceMinor: balanceMinor,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	accounts = append(accounts, account)
	if err := repository.FRepository.SaveAccounts(accounts); err != nil {
		return domain.Account{}, err
	}

	return account, nil
}

func UpdateAccountBalance(name, amountInput string) (domain.Account, error) {
	accountName := strings.TrimSpace(name)
	if accountName == "" {
		return domain.Account{}, fmt.Errorf("account name is required")
	}

	balanceMinor, err := ParseAmount(amountInput)
	if err != nil {
		return domain.Account{}, err
	}

	accounts, err := repository.FRepository.LoadAccounts()
	if err != nil {
		return domain.Account{}, err
	}

	for i := range accounts {
		if strings.EqualFold(accounts[i].Name, accountName) {
			accounts[i].BalanceMinor = balanceMinor
			accounts[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := repository.FRepository.SaveAccounts(accounts); err != nil {
				return domain.Account{}, err
			}
			return accounts[i], nil
		}
	}

	return domain.Account{}, fmt.Errorf("account '%s' not found", accountName)
}

func applyAccountDeltaToFile(accounts []domain.Account, accountName, currency string, deltaMinor int64, now string) error {
	name := strings.TrimSpace(accountName)
	cur := strings.ToUpper(strings.TrimSpace(currency))
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

func transferBetweenAccountsInFile(accounts []domain.Account, fromAccount, toAccount, currency string, amountMinor int64, now string) error {
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

	fromCurrency := strings.ToUpper(strings.TrimSpace(accounts[fromIdx].Currency))
	toCurrency := strings.ToUpper(strings.TrimSpace(accounts[toIdx].Currency))
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
