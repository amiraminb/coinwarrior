package tui

import (
	"slices"
	"strings"

	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/amiraminb/coinwarrior/internal/service"
)

var (
	Svc  *service.Service
	Repo repository.Repository
)

func loadCategoryChoices() ([]string, error) {
	categories, err := Svc.LoadCategories()
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(categories, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
	return categories, nil
}
