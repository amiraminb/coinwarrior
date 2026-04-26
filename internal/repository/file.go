package repository

import (
	"os"
	"path/filepath"
)

const (
	ConfigFileName       = "config.json"
	TransactionsFileName = "transactions.json"
	AccountsFileName     = "accounts.json"
	CategoriesFileName   = "categories.json"
	BudgetsFileName      = "budgets.json"
	RecurringFileName    = "recurring.json"
)

// DataDir returns the directory where coinwarrior stores local data files.
func (r *FileRepository) DataDir() (string, error) {
	if r.dataDir != "" {
		return r.dataDir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, "Documents", ".coinwarrior"), nil
}

// DataFilePath returns the absolute path for a coinwarrior data file.
func (r *FileRepository) DataFilePath(fileName string) (string, error) {
	dir, err := r.DataDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, fileName), nil
}

// CreateFile creates a data file with content if it does not already exist.
func (r *FileRepository) CreateFile(fileName string, content []byte) (string, bool, error) {
	path, err := r.DataFilePath(fileName)
	if err != nil {
		return "", false, err
	}

	if _, err := os.Stat(path); err == nil {
		return path, false, nil
	} else if !os.IsNotExist(err) {
		return path, false, err
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return path, false, err
	}

	return path, true, nil
}
