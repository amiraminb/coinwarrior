package repository

import "github.com/amiraminb/coinwarrior/internal/model"

type Repository interface {
	LoadAccounts() ([]model.Account, error)
	SaveAccounts([]model.Account) error
	LoadTransactions() ([]model.Transaction, error)
	SaveTransactions([]model.Transaction) error
	LoadCategories() ([]string, error)
	SaveCategories([]string) error
	LoadBudgets() ([]model.Budget, error)
	SaveBudgets([]model.Budget) error
	LoadRecurringRules() ([]model.RecurringRule, error)
	SaveRecurringRules([]model.RecurringRule) error
}

type FileRepository struct {
	dataDir string
}

func NewFileRepository() *FileRepository {
	return &FileRepository{}
}
