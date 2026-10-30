package service

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/amiraminb/coinwarrior/internal/model"
)

type categoryMutationRepo struct {
	budgetRepo
	failCategories bool
}

func (r *categoryMutationRepo) SaveCategories(categories []string) error {
	if r.failCategories {
		r.failCategories = false
		return errors.New("forced category save failure")
	}
	r.categories = slices.Clone(categories)
	return nil
}

func categoryManagementRepo() *categoryMutationRepo {
	r := &categoryMutationRepo{}
	r.categories = []string{"Groceries", "Dining", "Unused"}
	r.accounts = []model.Account{{Name: "Checking", Currency: "CAD", BalanceMinor: 10000}}
	r.transactions = []model.Transaction{
		{ID: "expense", Type: "expense", Category: "groceries", AmountMinor: 2000},
		{ID: "refund", Type: "income", Category: "Groceries", AmountMinor: 200},
		{ID: "other", Type: "expense", Category: "Dining", AmountMinor: 500},
		{ID: "transfer", Type: "transfer", Category: model.TransferCategory, AmountMinor: 1000},
	}
	r.rules = []model.RecurringRule{{ID: "rule", Type: "expense", Category: "Groceries", AmountMinor: 1000}}
	r.budgets = []model.Budget{
		{Month: "2026-11", Currency: "CAD", Category: "Groceries", AmountMinor: 50000},
		{Month: "2026-11", Currency: "CAD", AmountMinor: 350000},
	}
	return r
}

func TestRenameCategoryUpdatesReferencesAndPreservesFinancialValues(t *testing.T) {
	r := categoryManagementRepo()
	if err := New(r).RenameCategory(" groceries ", "Food Shopping"); err != nil {
		t.Fatal(err)
	}
	if r.categories[0] != "Food Shopping" || r.transactions[0].Category != "Food Shopping" ||
		r.transactions[1].Category != "Food Shopping" || r.rules[0].Category != "Food Shopping" || r.budgets[0].Category != "Food Shopping" {
		t.Fatal("rename did not update all matching references")
	}
	if r.transactions[2].Category != "Dining" || r.transactions[3].Category != model.TransferCategory || r.budgets[1].Category != "" {
		t.Fatal("rename affected unrelated categories")
	}
	if r.transactions[0].AmountMinor != 2000 || r.transactions[1].AmountMinor != 200 || r.accounts[0].BalanceMinor != 10000 || r.budgets[0].AmountMinor != 50000 {
		t.Fatal("rename changed financial values")
	}
	if r.transactions[0].UpdatedAt == "" || r.rules[0].UpdatedAt == "" || r.budgets[0].UpdatedAt == "" {
		t.Fatal("renamed records should have updated timestamps")
	}
}

func TestRenameCategoryRollsBackAllReferencesOnSaveFailure(t *testing.T) {
	r := categoryManagementRepo()
	before := categoryManagementRepo()
	r.failCategories = true
	if err := New(r).RenameCategory("Groceries", "Food Shopping"); err == nil {
		t.Fatal("expected save failure")
	}
	if !reflect.DeepEqual(r, before) {
		t.Fatal("failed rename left a partial update")
	}
}

func TestRenameCategoryRejectsCollisionsAndAllowsCasingChanges(t *testing.T) {
	r := categoryManagementRepo()
	s := New(r)
	for _, names := range [][2]string{{"Groceries", "dining"}, {"Groceries", " "}, {"Unknown", "New"}} {
		if err := s.RenameCategory(names[0], names[1]); err == nil {
			t.Errorf("rename %q to %q should fail", names[0], names[1])
		}
	}
	if err := s.RenameCategory("Groceries", "GROCERIES"); err != nil {
		t.Fatal(err)
	}
	if r.categories[0] != "GROCERIES" || r.transactions[0].Category != "GROCERIES" {
		t.Error("case-only rename should update the saved spelling")
	}
}

func TestDeleteCategoryChecksEveryReferenceType(t *testing.T) {
	for _, kind := range []string{"transaction", "recurring rule", "budget"} {
		t.Run(kind, func(t *testing.T) {
			r := &categoryMutationRepo{}
			r.categories = []string{"Groceries"}
			switch kind {
			case "transaction":
				r.transactions = []model.Transaction{{Type: "expense", Category: "groceries"}}
			case "recurring rule":
				r.rules = []model.RecurringRule{{Type: "expense", Category: "Groceries"}}
			case "budget":
				r.budgets = []model.Budget{{Category: "Groceries"}}
			}
			if err := New(r).DeleteCategory("Groceries"); err == nil {
				t.Fatal("referenced category should not be deleted")
			}
			if len(r.categories) != 1 {
				t.Fatal("failed deletion changed categories")
			}
		})
	}
	r := categoryManagementRepo()
	if err := New(r).DeleteCategory(" unused "); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.categories, []string{"Groceries", "Dining"}) {
		t.Errorf("unexpected remaining categories: %v", r.categories)
	}
}
