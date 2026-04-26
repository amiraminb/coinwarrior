package internal

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/amiraminb/coinwarrior/internal/domain"
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

	categoriesFile, err := loadCategoriesFile(path)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(categoriesFile.Categories))
	copy(result, categoriesFile.Categories)
	return result, nil
}

func AddCategory(category string) error {
	clean := strings.TrimSpace(category)
	if clean == "" {
		return nil
	}

	path, err := FilePath(CategoriesFileName)
	if err != nil {
		return err
	}

	categoriesFile, err := loadCategoriesFile(path)
	if err != nil {
		return err
	}

	for _, existing := range categoriesFile.Categories {
		if strings.EqualFold(existing, clean) {
			return nil
		}
	}

	categoriesFile.Categories = append(categoriesFile.Categories, clean)
	return saveCategoriesFile(path, categoriesFile)
}

func loadCategoriesFile(path string) (domain.CategoriesFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			result := make([]string, len(DefaultCategories))
			copy(result, DefaultCategories)
			return domain.CategoriesFile{SchemaVersion: 1, Categories: result}, nil
		}
		return domain.CategoriesFile{}, err
	}

	var categoriesFile domain.CategoriesFile
	if err := json.Unmarshal(data, &categoriesFile); err != nil {
		return domain.CategoriesFile{}, err
	}

	if categoriesFile.Categories == nil {
		categoriesFile.Categories = []string{}
	}

	return categoriesFile, nil
}

func saveCategoriesFile(path string, categoriesFile domain.CategoriesFile) error {
	if categoriesFile.Categories == nil {
		categoriesFile.Categories = []string{}
	}

	data, err := json.MarshalIndent(categoriesFile, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

func CategoryExists(categories []string, category string) bool {
	for _, existing := range categories {
		if strings.EqualFold(existing, category) {
			return true
		}
	}
	return false
}
