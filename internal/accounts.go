package internal

import "strings"

func LoadAccounts() ([]string, error) {
	path, err := FilePath(TransactionsFileName)
	if err != nil {
		return nil, err
	}

	transactions, err := LoadTransactions(path)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	for _, tx := range transactions.Transactions {
		result = append(result, tx.Account)
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
