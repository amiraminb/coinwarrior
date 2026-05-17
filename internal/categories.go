package internal

import (
	"slices"
	"strings"

	"github.com/amiraminb/coinwarrior/internal/repository"
)

func LoadCategories() ([]string, error) {
	categories, err := repository.FRepository.LoadCategories()
	if err != nil {
		return nil, err
	}

	result := make([]string, len(categories))
	copy(result, categories)
	return result, nil
}

func AddCategory(category string) error {
	clean := strings.TrimSpace(category)
	if clean == "" {
		return nil
	}

	categories, err := repository.FRepository.LoadCategories()
	if err != nil {
		return err
	}

	if slices.ContainsFunc(categories, func(s string) bool { return strings.EqualFold(s, clean) }) {
		return nil
	}

	categories = append(categories, clean)
	return repository.FRepository.SaveCategories(categories)
}

