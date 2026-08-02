package tui

import (
	"strings"
	"testing"

	"github.com/amiraminb/coinwarrior/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

func keyPress(name string) tea.KeyMsg {
	switch name {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	}
	return keyRunes(name)
}

func advance(m editModel, keys ...string) editModel {
	for _, k := range keys {
		next, _ := m.Update(keyPress(k))
		m = next.(editModel)
	}
	return m
}

func expenseTx(category string) model.Transaction {
	return model.Transaction{
		ID:          "txn_1",
		Type:        model.TransactionTypeExpense,
		AmountMinor: 2000,
		Currency:    "CAD",
		Date:        "2026-08-01",
		Category:    category,
		Account:     "Chk",
	}
}

// The cursor must open on the transaction's own category so pressing enter is a
// no-op rather than silently reassigning it to the first list entry.
func TestEditCategoryCursorStartsOnTheCurrentCategory(t *testing.T) {
	categories := []string{"Housing", "Groceries", "Dining"}

	m := newEditModel(expenseTx("Dining"), categories)

	if m.categoryCursor != 2 {
		t.Errorf("categoryCursor = %d, want 2 (Dining)", m.categoryCursor)
	}
}

func TestEditCategoryCursorMatchesCaseInsensitively(t *testing.T) {
	m := newEditModel(expenseTx("groceries"), []string{"Housing", "Groceries"})

	if m.categoryCursor != 1 {
		t.Errorf("categoryCursor = %d, want 1 despite differing case", m.categoryCursor)
	}
}

// A category missing from categories.json must still be listed and pre-selected,
// or a single enter would silently reassign the transaction.
func TestEditUnlistedCategoryIsListedAndPreselected(t *testing.T) {
	m := newEditModel(expenseTx("Retired"), []string{"Housing", "Groceries"})
	m.step = editStepCategorySelect

	if m.categories[0] != "Retired" {
		t.Errorf("categories[0] = %q, want the unlisted current category first", m.categories[0])
	}
	if m.categoryCursor != 0 {
		t.Errorf("categoryCursor = %d, want 0", m.categoryCursor)
	}

	m = advance(m, "enter")
	if m.categoryInput != "Retired" {
		t.Errorf("one enter changed the category to %q; it must stay %q", m.categoryInput, "Retired")
	}
}

func TestEditUncategorizedStaysUncategorizedOnEnter(t *testing.T) {
	m := newEditModel(expenseTx(""), []string{"Housing", "Groceries"})
	m.step = editStepCategorySelect

	if m.categories[0] != noCategoryLabel {
		t.Errorf("categories[0] = %q, want %q", m.categories[0], noCategoryLabel)
	}

	m = advance(m, "enter")
	if m.categoryInput != "" {
		t.Errorf("one enter set the category to %q; an uncategorized transaction must stay empty", m.categoryInput)
	}
}

// Typing an existing name in another case must store the saved spelling, so the
// ledger and categories.json cannot drift apart.
func TestEditTypedCategoryAdoptsTheStoredCasing(t *testing.T) {
	m := newEditModel(expenseTx("Housing"), []string{"Housing", "Groceries"})
	m.step = editStepCategoryInput

	m = advance(m, "g", "r", "o", "c", "e", "r", "i", "e", "s", "enter")

	if m.categoryInput != "Groceries" {
		t.Errorf("categoryInput = %q, want the stored casing %q", m.categoryInput, "Groceries")
	}
	if m.step != editStepAccount {
		t.Errorf("step = %d, want editStepAccount", m.step)
	}
}

func TestEditSelectingACategoryAdvancesToAccount(t *testing.T) {
	m := newEditModel(expenseTx("Housing"), []string{"Housing", "Groceries"})
	m.step = editStepCategorySelect

	m = advance(m, "down", "enter")

	if m.categoryInput != "Groceries" {
		t.Errorf("categoryInput = %q, want %q", m.categoryInput, "Groceries")
	}
	if m.step != editStepAccount {
		t.Errorf("step = %d, want editStepAccount", m.step)
	}
}

func TestEditCursorStopsAtTheNewCategoryEntry(t *testing.T) {
	categories := []string{"Housing", "Groceries"}
	m := newEditModel(expenseTx("Housing"), categories)
	m.step = editStepCategorySelect

	m = advance(m, "down", "down", "down", "down")

	if m.categoryCursor != len(categories) {
		t.Errorf("categoryCursor = %d, want %d ([New category])", m.categoryCursor, len(categories))
	}

	m = advance(m, "enter")
	if m.step != editStepCategoryInput {
		t.Errorf("step = %d, want editStepCategoryInput", m.step)
	}
}

func TestEditNewCategoryNeedsConfirmation(t *testing.T) {
	m := newEditModel(expenseTx("Housing"), []string{"Housing"})
	m.step = editStepCategoryInput

	m = advance(m, "S", "p", "a", "enter")

	if m.step != editStepCategoryConfirm {
		t.Fatalf("step = %d, want editStepCategoryConfirm", m.step)
	}
	if m.pendingCategory != "Spa" {
		t.Errorf("pendingCategory = %q, want %q", m.pendingCategory, "Spa")
	}

	m = advance(m, "enter")
	if m.categoryInput != "Spa" {
		t.Errorf("categoryInput = %q, want %q after confirming", m.categoryInput, "Spa")
	}
	if m.step != editStepAccount {
		t.Errorf("step = %d, want editStepAccount", m.step)
	}
}

// Re-typing an existing name must not ask to create it again.
func TestEditTypingAnExistingCategorySkipsConfirmation(t *testing.T) {
	m := newEditModel(expenseTx("Housing"), []string{"Housing", "Groceries"})
	m.step = editStepCategoryInput

	m = advance(m, "G", "r", "o", "c", "e", "r", "i", "e", "s", "enter")

	if m.step != editStepAccount {
		t.Errorf("step = %d, want editStepAccount (no confirm for a known name)", m.step)
	}
	if m.categoryInput != "Groceries" {
		t.Errorf("categoryInput = %q, want %q", m.categoryInput, "Groceries")
	}
}

func TestEditEmptyCategoryDraftIsRejected(t *testing.T) {
	m := newEditModel(expenseTx("Housing"), []string{"Housing"})
	m.step = editStepCategoryInput

	m = advance(m, "enter")

	if m.step != editStepCategoryInput {
		t.Errorf("step = %d, want to stay on editStepCategoryInput", m.step)
	}
	if m.errMessage == "" {
		t.Error("expected an error message for an empty category")
	}
}

// A transfer's category is always model.TransferCategory and is excluded from
// every category report, so the picker must not appear for one.
func TestEditTransferSkipsTheCategorySteps(t *testing.T) {
	tx := expenseTx(model.TransferCategory)
	tx.Type = model.TransactionTypeTransfer
	tx.ToAccount = "Savings"

	m := newEditModel(tx, []string{"Housing"})
	m.step = editStepAmount

	m = advance(m, "enter")

	if m.step != editStepAccount {
		t.Errorf("step = %d, want editStepAccount (category skipped for a transfer)", m.step)
	}

	m = advance(m, "esc")
	if m.step != editStepAmount {
		t.Errorf("esc from account went to step %d, want editStepAmount for a transfer", m.step)
	}
}

func TestEditNonTransferGoesBackToThePicker(t *testing.T) {
	m := newEditModel(expenseTx("Housing"), []string{"Housing"})
	m.step = editStepAccount

	m = advance(m, "esc")

	if m.step != editStepCategorySelect {
		t.Errorf("step = %d, want editStepCategorySelect", m.step)
	}
}

func TestEditCategoryViewListsEveryCategory(t *testing.T) {
	categories := []string{"Housing", "Groceries", "Dining"}
	m := newEditModel(expenseTx("Groceries"), categories)
	m.step = editStepCategorySelect

	view := m.View()

	for _, c := range categories {
		if !strings.Contains(view, c) {
			t.Errorf("view omits category %q:\n%s", c, view)
		}
	}
	if !strings.Contains(view, "[New category]") {
		t.Errorf("view omits the new-category entry:\n%s", view)
	}
}

func TestEditCategoryViewLabelsAnEmptyCurrentCategory(t *testing.T) {
	m := newEditModel(expenseTx(""), []string{"Housing"})
	m.step = editStepCategorySelect

	if !strings.Contains(m.View(), "(no category)") {
		t.Errorf("an uncategorized transaction should show (no category):\n%s", m.View())
	}
}

// A category literally named "(no category)" must survive: the clear-category
// row is identified by position, not by matching that display string.
func TestEditCategoryNamedLikeTheNoCategoryLabelSurvives(t *testing.T) {
	m := newEditModel(expenseTx(noCategoryLabel), []string{"Housing"})
	m.step = editStepCategorySelect

	m = advance(m, "enter")

	if m.categoryInput != noCategoryLabel {
		t.Errorf("categoryInput = %q, want %q preserved", m.categoryInput, noCategoryLabel)
	}
}

func TestEditPickingASavedCategoryNamedLikeTheLabelKeepsIt(t *testing.T) {
	m := newEditModel(expenseTx("Housing"), []string{"Housing", noCategoryLabel})
	m.step = editStepCategorySelect

	m = advance(m, "down", "enter")

	if m.categoryInput != noCategoryLabel {
		t.Errorf("categoryInput = %q, want the saved category %q", m.categoryInput, noCategoryLabel)
	}
}

func TestEditClearRowIndexIsOnlySetForAnUncategorizedTransaction(t *testing.T) {
	uncategorized := newEditModel(expenseTx(""), []string{"Housing"})
	if uncategorized.clearCategoryIndex != 0 {
		t.Errorf("clearCategoryIndex = %d, want 0", uncategorized.clearCategoryIndex)
	}

	for _, stored := range []string{"Housing", "Retired", noCategoryLabel} {
		m := newEditModel(expenseTx(stored), []string{"Housing"})
		if m.clearCategoryIndex != -1 {
			t.Errorf("stored %q: clearCategoryIndex = %d, want -1", stored, m.clearCategoryIndex)
		}
	}
}

func TestEditCategoriesEmptyListStillOffersTheCurrentCategory(t *testing.T) {
	m := newEditModel(expenseTx("Groceries"), nil)
	m.step = editStepCategorySelect

	if len(m.categories) != 1 || m.categories[0] != "Groceries" {
		t.Fatalf("categories = %q, want just the current category", m.categories)
	}

	m = advance(m, "enter")
	if m.categoryInput != "Groceries" {
		t.Errorf("categoryInput = %q, want %q with an empty saved list", m.categoryInput, "Groceries")
	}
}

func TestEditWhitespacePaddedCategoryIsNotDuplicated(t *testing.T) {
	m := newEditModel(expenseTx("  Groceries  "), []string{"Housing", "Groceries"})

	if len(m.categories) != 2 {
		t.Errorf("categories = %q, want no duplicate row for a padded stored value", m.categories)
	}
	if m.categoryCursor != 1 {
		t.Errorf("categoryCursor = %d, want 1 (the existing Groceries row)", m.categoryCursor)
	}
}
