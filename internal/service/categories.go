package service

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
)

func (s *Service) LoadCategories() ([]string, error) {
	categories, err := s.repo.LoadCategories()
	if err != nil {
		return nil, err
	}

	result := make([]string, len(categories))
	copy(result, categories)
	return result, nil
}

func (s *Service) ResolveCategory(name string) (string, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return "", errors.New("category cannot be empty")
	}

	known, err := s.LoadCategories()
	if err != nil {
		return "", err
	}

	if i := slices.IndexFunc(known, func(c string) bool { return strings.EqualFold(c, clean) }); i >= 0 {
		return known[i], nil
	}

	return "", fmt.Errorf("unknown category '%s' (available: %s)", clean, strings.Join(known, ", "))
}

func (s *Service) AddCategory(category string) error {
	clean := strings.TrimSpace(category)
	if clean == "" {
		return nil
	}

	categories, err := s.repo.LoadCategories()
	if err != nil {
		return err
	}

	if slices.ContainsFunc(categories, func(s string) bool { return strings.EqualFold(s, clean) }) {
		return nil
	}

	categories = append(categories, clean)
	return s.repo.SaveCategories(categories)
}

func categoryIndex(categories []string, name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return -1, errors.New("category cannot be empty")
	}
	i := slices.IndexFunc(categories, func(c string) bool { return strings.EqualFold(c, name) })
	if i < 0 {
		return -1, fmt.Errorf("unknown category '%s' (available: %s)", name, strings.Join(categories, ", "))
	}
	return i, nil
}

// RenameCategory updates every reference to a saved category. Financial amounts
// and account balances are independent of category names.
func (s *Service) RenameCategory(name, newName string) error {
	categories, err := s.LoadCategories()
	if err != nil {
		return err
	}
	i, err := categoryIndex(categories, name)
	if err != nil {
		return err
	}
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return errors.New("new category cannot be empty")
	}
	if existing, err := categoryIndex(categories, newName); err == nil && existing != i {
		return fmt.Errorf("category '%s' already exists", categories[existing])
	}
	oldName := categories[i]
	if oldName == newName {
		return nil
	}

	transactions, err := s.repo.LoadTransactions()
	if err != nil {
		return err
	}
	rules, err := s.repo.LoadRecurringRules()
	if err != nil {
		return err
	}
	budgets, err := s.repo.LoadBudgets()
	if err != nil {
		return err
	}
	originalCategories := slices.Clone(categories)
	originalTransactions := cloneTransactions(transactions)
	originalRules := slices.Clone(rules)
	originalBudgets := slices.Clone(budgets)
	transactions = cloneTransactions(transactions)
	rules = slices.Clone(rules)
	budgets = slices.Clone(budgets)
	now := time.Now().UTC().Format(time.RFC3339)
	matches := func(category string) bool {
		return strings.EqualFold(strings.TrimSpace(category), oldName)
	}

	type change struct {
		save    func() error
		restore func() error
	}
	var changes []change
	changed := false
	for i := range transactions {
		if transactions[i].Type != model.TransactionTypeTransfer && matches(transactions[i].Category) {
			transactions[i].Category = newName
			transactions[i].UpdatedAt = now
			changed = true
		}
	}
	if changed {
		changes = append(changes, change{
			func() error { return s.repo.SaveTransactions(transactions) },
			func() error { return s.repo.SaveTransactions(originalTransactions) },
		})
	}
	changed = false
	for i := range rules {
		if rules[i].Type != model.TransactionTypeTransfer && matches(rules[i].Category) {
			rules[i].Category = newName
			rules[i].UpdatedAt = now
			changed = true
		}
	}
	if changed {
		changes = append(changes, change{
			func() error { return s.repo.SaveRecurringRules(rules) },
			func() error { return s.repo.SaveRecurringRules(originalRules) },
		})
	}
	changed = false
	for i := range budgets {
		if matches(budgets[i].Category) {
			budgets[i].Category = newName
			budgets[i].UpdatedAt = now
			changed = true
		}
	}
	if changed {
		changes = append(changes, change{
			func() error { return s.repo.SaveBudgets(budgets) },
			func() error { return s.repo.SaveBudgets(originalBudgets) },
		})
	}
	categories[i] = newName
	changes = append(changes, change{
		func() error { return s.repo.SaveCategories(categories) },
		func() error { return s.repo.SaveCategories(originalCategories) },
	})
	for i, change := range changes {
		if err := change.save(); err != nil {
			var restores []func() error
			for j := i; j >= 0; j-- {
				restores = append(restores, changes[j].restore)
			}
			return s.rollback(err, restores...)
		}
	}
	return nil
}

func (s *Service) DeleteCategory(name string) error {
	categories, err := s.LoadCategories()
	if err != nil {
		return err
	}
	i, err := categoryIndex(categories, name)
	if err != nil {
		return err
	}
	name = categories[i]
	transactions, err := s.repo.LoadTransactions()
	if err != nil {
		return err
	}
	rules, err := s.repo.LoadRecurringRules()
	if err != nil {
		return err
	}
	budgets, err := s.repo.LoadBudgets()
	if err != nil {
		return err
	}
	txCount, ruleCount, budgetCount := 0, 0, 0
	for _, tx := range transactions {
		if tx.Type != model.TransactionTypeTransfer && strings.EqualFold(strings.TrimSpace(tx.Category), name) {
			txCount++
		}
	}
	for _, rule := range rules {
		if rule.Type != model.TransactionTypeTransfer && strings.EqualFold(strings.TrimSpace(rule.Category), name) {
			ruleCount++
		}
	}
	for _, budget := range budgets {
		if strings.EqualFold(budget.Category, name) {
			budgetCount++
		}
	}
	if txCount+ruleCount+budgetCount > 0 {
		return fmt.Errorf("category '%s' is used by %d transactions, %d recurring rules, and %d budgets; reassign them before deleting it", name, txCount, ruleCount, budgetCount)
	}
	return s.repo.SaveCategories(slices.Delete(categories, i, i+1))
}
