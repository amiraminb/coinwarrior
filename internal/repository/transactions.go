package repository

import "github.com/amiraminb/coinwarrior/internal/model"

type transactionsDocument struct {
	SchemaVersion int                 `json:"schema_version"`
	Transactions  []model.Transaction `json:"transactions"`
}

func (r *FileRepository) LoadTransactions() ([]model.Transaction, error) {
	path, err := r.DataFilePath(TransactionsFileName)
	if err != nil {
		return nil, err
	}
	return loadDocument(path, func(d transactionsDocument) []model.Transaction {
		return d.Transactions
	}, nil)
}

func (r *FileRepository) SaveTransactions(transactions []model.Transaction) error {
	path, err := r.DataFilePath(TransactionsFileName)
	if err != nil {
		return err
	}
	if transactions == nil {
		transactions = []model.Transaction{}
	}
	return saveDocument(path, transactionsDocument{SchemaVersion: 1, Transactions: transactions})
}
