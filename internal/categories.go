package internal

import "strings"

func LoadCategories() ([]string, error) {
	path, err := FilePath(TransactionsFileName)
	if err != nil {
		return nil, err
	}

	file, err := LoadTransactions(path)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	for _, tx := range file.Transactions {
		result = append(result, tx.Category)
	}

	return result, nil
}

func CategoryExists(categories []string, category string) bool {
	for _, existing := range categories {
		if strings.EqualFold(existing, category) {
			return true
		}
	}
	return false
}
