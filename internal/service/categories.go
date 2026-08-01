package service

import (
	"errors"
	"fmt"
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
