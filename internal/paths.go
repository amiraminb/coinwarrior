package internal

import (
	"os"
	"path/filepath"
)

const (
	ConfigFileName       = "config.json"
	TransactionsFileName = "transactions.json"
	BudgetsFileName      = "budgets.json"
	RecurringFileName    = "recurring.json"
)

func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, "Documents", ".coinwarrior"), nil
}

func FilePath(name string) (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, name), nil
}
