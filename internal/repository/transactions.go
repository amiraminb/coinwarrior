package repository

import "github.com/amiraminb/coinwarrior/internal/domain"

type transactionsDocument struct {
	SchemaVersion int                  `json:"schema_version"`
	Transactions  []domain.Transaction `json:"transactions"`
}

func (r *FileRepository) LoadTransactions() ([]domain.Transaction, error) {
	path, err := r.DataFilePath(TransactionsFileName)
	if err != nil {
		return nil, err
	}
	return loadDocument(path, func(d transactionsDocument) []domain.Transaction {
		return d.Transactions
	}, nil)
}

func (r *FileRepository) SaveTransactions(transactions []domain.Transaction) error {
	path, err := r.DataFilePath(TransactionsFileName)
	if err != nil {
		return err
	}
	if transactions == nil {
		transactions = []domain.Transaction{}
	}
	return saveDocument(path, transactionsDocument{SchemaVersion: 1, Transactions: transactions})
}
