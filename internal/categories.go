package internal

import (
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

	for _, existing := range categories {
		if strings.EqualFold(existing, clean) {
			return nil
		}
	}

	categories = append(categories, clean)
	return repository.FRepository.SaveCategories(categories)
}

func CategoryExists(categories []string, category string) bool {
	for _, existing := range categories {
		if strings.EqualFold(existing, category) {
			return true
		}
	}
	return false
}
