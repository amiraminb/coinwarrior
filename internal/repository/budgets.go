package repository

import (
	"encoding/json"
	"os"

	"github.com/amiraminb/coinwarrior/internal/domain"
)

type budgetsDocument struct {
	SchemaVersion int             `json:"schema_version"`
	Budgets       []domain.Budget `json:"budgets"`
}

func (r *FileRepository) LoadBudgets() ([]domain.Budget, error) {
	path, err := r.DataFilePath(BudgetsFileName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.Budget{}, nil
		}
		return nil, err
	}

	var document budgetsDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	if document.Budgets == nil {
		document.Budgets = []domain.Budget{}
	}

	return document.Budgets, nil
}

func (r *FileRepository) SaveBudgets(budgets []domain.Budget) error {
	path, err := r.DataFilePath(BudgetsFileName)
	if err != nil {
		return err
	}

	if budgets == nil {
		budgets = []domain.Budget{}
	}

	data, err := json.MarshalIndent(budgetsDocument{SchemaVersion: 1, Budgets: budgets}, "", "  ")
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
