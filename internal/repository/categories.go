package repository

import (
	"encoding/json"
	"os"

	"github.com/amiraminb/coinwarrior/internal/domain"
)

type categoriesDocument struct {
	SchemaVersion int      `json:"schema_version"`
	Categories    []string `json:"categories"`
}

func (r *FileRepository) LoadCategories() ([]string, error) {
	path, err := r.DataFilePath(CategoriesFileName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			categories := make([]string, len(domain.DefaultCategories))
			copy(categories, domain.DefaultCategories)
			return categories, nil
		}
		return nil, err
	}

	var document categoriesDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	if document.Categories == nil {
		document.Categories = []string{}
	}

	return document.Categories, nil
}

func (r *FileRepository) SaveCategories(categories []string) error {
	path, err := r.DataFilePath(CategoriesFileName)
	if err != nil {
		return err
	}

	if categories == nil {
		categories = []string{}
	}

	data, err := json.MarshalIndent(categoriesDocument{SchemaVersion: 1, Categories: categories}, "", "  ")
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
