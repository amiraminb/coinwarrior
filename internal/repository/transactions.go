package repository

import (
	"encoding/json"
	"os"

	"github.com/amiraminb/coinwarrior/internal/domain"
)

type transactionsDocument struct {
	SchemaVersion int                  `json:"schema_version"`
	Transactions  []domain.Transaction `json:"transactions"`
}

func (r *FileRepository) LoadTransactions() ([]domain.Transaction, error) {
	path, err := r.DataFilePath(TransactionsFileName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.Transaction{}, nil
		}
		return nil, err
	}

	var document transactionsDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	if document.Transactions == nil {
		document.Transactions = []domain.Transaction{}
	}

	return document.Transactions, nil
}

func (r *FileRepository) SaveTransactions(transactions []domain.Transaction) error {
	path, err := r.DataFilePath(TransactionsFileName)
	if err != nil {
		return err
	}

	if transactions == nil {
		transactions = []domain.Transaction{}
	}

	data, err := json.MarshalIndent(transactionsDocument{SchemaVersion: 1, Transactions: transactions}, "", "  ")
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
