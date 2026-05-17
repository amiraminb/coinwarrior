package repository

import "github.com/amiraminb/coinwarrior/internal/domain"

type accountsDocument struct {
	SchemaVersion int              `json:"schema_version"`
	Accounts      []domain.Account `json:"accounts"`
}

func (r *FileRepository) LoadAccounts() ([]domain.Account, error) {
	path, err := r.DataFilePath(AccountsFileName)
	if err != nil {
		return nil, err
	}
	return loadDocument(path, func(d accountsDocument) []domain.Account {
		return d.Accounts
	}, nil)
}

func (r *FileRepository) SaveAccounts(accounts []domain.Account) error {
	path, err := r.DataFilePath(AccountsFileName)
	if err != nil {
		return err
	}
	if accounts == nil {
		accounts = []domain.Account{}
	}
	return saveDocument(path, accountsDocument{SchemaVersion: 1, Accounts: accounts})
}
