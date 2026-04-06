package internal

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/amiraminb/coinwarrior/internal/model"
)

var DefaultCategories = []string{
	"Housing",
	"Utilities",
	"Groceries",
	"Dining",
	"Transportation",
	"Healthcare",
	"Insurance",
	"Subscriptions",
	"Entertainment",
	"Income",
}

func LoadCategories() ([]string, error) {
	path, err := FilePath(CategoriesFileName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			result := make([]string, len(DefaultCategories))
			copy(result, DefaultCategories)
			return result, nil
		}
		return nil, err
	}

	var categoriesFile model.CategoriesFile
	if err := json.Unmarshal(data, &categoriesFile); err != nil {
		return nil, err
	}

	if categoriesFile.Categories == nil {
		return []string{}, nil
	}

	result := make([]string, len(categoriesFile.Categories))
	copy(result, categoriesFile.Categories)
	return result, nil
}

func CategoryExists(categories []string, category string) bool {
	for _, existing := range categories {
		if strings.EqualFold(existing, category) {
			return true
		}
	}
	return false
}
