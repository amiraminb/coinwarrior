package service

import (
	"slices"
	"strings"
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

