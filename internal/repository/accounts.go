package repository

import (
	"encoding/json"
	"os"

	"github.com/amiraminb/coinwarrior/internal/domain"
)

type accountsDocument struct {
	SchemaVersion int              `json:"schema_version"`
	Accounts      []domain.Account `json:"accounts"`
}

func (r *FileRepository) LoadAccounts() ([]domain.Account, error) {
	path, err := r.DataFilePath(AccountsFileName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.Account{}, nil
		}
		return nil, err
	}

	var document accountsDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	if document.Accounts == nil {
		document.Accounts = []domain.Account{}
	}

	return document.Accounts, nil
}

func (r *FileRepository) SaveAccounts(accounts []domain.Account) error {
	path, err := r.DataFilePath(AccountsFileName)
	if err != nil {
		return err
	}

	if accounts == nil {
		accounts = []domain.Account{}
	}

	data, err := json.MarshalIndent(accountsDocument{SchemaVersion: 1, Accounts: accounts}, "", "  ")
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
