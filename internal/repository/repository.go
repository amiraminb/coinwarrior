package repository

import "github.com/amiraminb/coinwarrior/internal/domain"

type Repository interface {
	LoadAccounts() ([]domain.Account, error)
	SaveAccounts([]domain.Account) error
	LoadTransactions() ([]domain.Transaction, error)
	SaveTransactions([]domain.Transaction) error
	LoadCategories() ([]string, error)
	SaveCategories([]string) error
	LoadBudgets() ([]domain.Budget, error)
	SaveBudgets([]domain.Budget) error
}

type FileRepository struct {
	dataDir string
}

func NewFileRepository() *FileRepository {
	return &FileRepository{}
}

var FRepository Repository = NewFileRepository()
