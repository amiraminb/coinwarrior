package repository

import "github.com/amiraminb/coinwarrior/internal/domain"

type budgetsDocument struct {
	SchemaVersion int             `json:"schema_version"`
	Budgets       []domain.Budget `json:"budgets"`
}

func (r *FileRepository) LoadBudgets() ([]domain.Budget, error) {
	path, err := r.DataFilePath(BudgetsFileName)
	if err != nil {
		return nil, err
	}
	return loadDocument(path, func(d budgetsDocument) []domain.Budget {
		return d.Budgets
	}, nil)
}

func (r *FileRepository) SaveBudgets(budgets []domain.Budget) error {
	path, err := r.DataFilePath(BudgetsFileName)
	if err != nil {
		return err
	}
	if budgets == nil {
		budgets = []domain.Budget{}
	}
	return saveDocument(path, budgetsDocument{SchemaVersion: 1, Budgets: budgets})
}
