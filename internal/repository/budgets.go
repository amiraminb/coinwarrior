package repository

import "github.com/amiraminb/coinwarrior/internal/model"

type budgetsDocument struct {
	SchemaVersion int             `json:"schema_version"`
	Budgets       []model.Budget `json:"budgets"`
}

func (r *FileRepository) LoadBudgets() ([]model.Budget, error) {
	path, err := r.DataFilePath(BudgetsFileName)
	if err != nil {
		return nil, err
	}
	return loadDocument(path, func(d budgetsDocument) []model.Budget {
		return d.Budgets
	}, nil)
}

func (r *FileRepository) SaveBudgets(budgets []model.Budget) error {
	path, err := r.DataFilePath(BudgetsFileName)
	if err != nil {
		return err
	}
	if budgets == nil {
		budgets = []model.Budget{}
	}
	return saveDocument(path, budgetsDocument{SchemaVersion: 1, Budgets: budgets})
}
