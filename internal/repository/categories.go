package repository

import "github.com/amiraminb/coinwarrior/internal/model"

type categoriesDocument struct {
	SchemaVersion int      `json:"schema_version"`
	Categories    []string `json:"categories"`
}

func (r *FileRepository) LoadCategories() ([]string, error) {
	path, err := r.DataFilePath(CategoriesFileName)
	if err != nil {
		return nil, err
	}
	return loadDocument(path, func(d categoriesDocument) []string {
		return d.Categories
	}, func() []string {
		categories := make([]string, len(model.DefaultCategories))
		copy(categories, model.DefaultCategories)
		return categories
	})
}

func (r *FileRepository) SaveCategories(categories []string) error {
	path, err := r.DataFilePath(CategoriesFileName)
	if err != nil {
		return err
	}
	if categories == nil {
		categories = []string{}
	}
	return saveDocument(path, categoriesDocument{SchemaVersion: 1, Categories: categories})
}
