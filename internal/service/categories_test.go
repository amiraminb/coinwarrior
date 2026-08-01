package service

import (
	"strings"
	"testing"

	"github.com/amiraminb/coinwarrior/internal/model"
)

type categoriesRepo struct {
	fakeRepo
	categories []string
}

func (r *categoriesRepo) LoadCategories() ([]string, error) {
	out := make([]string, len(r.categories))
	copy(out, r.categories)
	return out, nil
}

func newCategoriesService(categories ...string) *Service {
	return New(&categoriesRepo{categories: categories})
}

func TestResolveCategoryMatchesCaseInsensitively(t *testing.T) {
	svc := newCategoriesService("Groceries", "Dining")

	for _, input := range []string{"Groceries", "groceries", "GROCERIES", "  groceries  "} {
		got, err := svc.ResolveCategory(input)
		if err != nil {
			t.Fatalf("ResolveCategory(%q): unexpected error: %v", input, err)
		}
		if got != "Groceries" {
			t.Errorf("ResolveCategory(%q) = %q, want %q", input, got, "Groceries")
		}
	}
}

func TestResolveCategoryAcceptsTransferWhenNotStored(t *testing.T) {
	svc := newCategoriesService("Groceries")

	got, err := svc.ResolveCategory("transfer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != model.TransferCategory {
		t.Errorf("got %q, want %q", got, model.TransferCategory)
	}
}

func TestResolveCategoryRejectsUnknownAndListsAvailable(t *testing.T) {
	svc := newCategoriesService("Groceries", "Dining")

	_, err := svc.ResolveCategory("Grocerys")
	if err == nil {
		t.Fatal("expected an error for an unknown category")
	}
	for _, want := range []string{"Grocerys", "Groceries", "Dining", model.TransferCategory} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}
}

func TestResolveCategoryRejectsEmpty(t *testing.T) {
	svc := newCategoriesService("Groceries")

	if _, err := svc.ResolveCategory("   "); err == nil {
		t.Fatal("expected an error for an empty category")
	}
}
