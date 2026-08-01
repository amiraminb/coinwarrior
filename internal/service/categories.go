package service

import (
	"fmt"
	"slices"
	"strings"

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

// Transfers carry model.TransferCategory without it ever being saved to
// categories.json, so it is accepted even when absent from the stored list.
func (s *Service) ResolveCategory(name string) (string, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return "", fmt.Errorf("category cannot be empty")
	}

	known, err := s.LoadCategories()
	if err != nil {
		return "", err
	}

	if !slices.ContainsFunc(known, func(c string) bool { return strings.EqualFold(c, model.TransferCategory) }) {
		known = append(known, model.TransferCategory)
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
