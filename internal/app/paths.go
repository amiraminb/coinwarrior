package app

import (
	"os"
	"path/filepath"
	"strings"
)

const DataDirEnv = "COINWARRIOR_DATA_DIR"

const (
	ConfigFileName       = "config.json"
	TransactionsFileName = "transactions.json"
	BudgetsFileName      = "budgets.json"
	RecurringFileName    = "recurring.json"
)

func DataDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv(DataDirEnv)); dir != "" {
		return dir, nil
	}

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
