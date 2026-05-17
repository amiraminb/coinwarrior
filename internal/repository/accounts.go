package repository

import "github.com/amiraminb/coinwarrior/internal/model"

type accountsDocument struct {
	SchemaVersion int              `json:"schema_version"`
	Accounts      []model.Account `json:"accounts"`
}

func (r *FileRepository) LoadAccounts() ([]model.Account, error) {
	path, err := r.DataFilePath(AccountsFileName)
	if err != nil {
		return nil, err
	}
	return loadDocument(path, func(d accountsDocument) []model.Account {
		return d.Accounts
	}, nil)
}

func (r *FileRepository) SaveAccounts(accounts []model.Account) error {
	path, err := r.DataFilePath(AccountsFileName)
	if err != nil {
		return err
	}
	if accounts == nil {
		accounts = []model.Account{}
	}
	return saveDocument(path, accountsDocument{SchemaVersion: 1, Accounts: accounts})
}
