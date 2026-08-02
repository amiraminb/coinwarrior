package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/amiraminb/coinwarrior/internal/importer"
)

func TestValidateAmountRejectsWhatTheImporterLetsThrough(t *testing.T) {
	for _, amount := range []string{"$45.67", "1.234", "45.67 CR", "", "abc", "-5.00", "0.00"} {
		if err := validateAmount(amount); err == nil {
			t.Errorf("validateAmount(%q) = nil; a row carrying it would fail mid-save", amount)
		}
	}
}

func TestValidateAmountAcceptsBankFormattedValues(t *testing.T) {
	for _, amount := range []string{"45.67", "1", "1.5", "1000.00"} {
		if err := validateAmount(amount); err != nil {
			t.Errorf("validateAmount(%q) = %v; want nil", amount, err)
		}
	}
}

// An unparseable amount must be caught before the save path runs, or the
// duplicate lookup fails and takes the whole import down with it.
func TestRowSaveBlockerCatchesAnUnparseableAmount(t *testing.T) {
	for _, amount := range []string{"$45.67", "1.234", "45.67 CR"} {
		row := importer.ParsedRow{Date: "2026-08-01", Type: "expense", AmountInput: amount}
		blocker := rowSaveBlocker(row, "Groceries")
		if blocker == "" {
			t.Errorf("rowSaveBlocker with amount %q returned no blocker; the save would proceed", amount)
			continue
		}
		if !strings.Contains(blocker, "fix the row") {
			t.Errorf("amount %q: unexpected blocker %q", amount, blocker)
		}
	}
}

func TestRowSaveBlockerReportsParseErrorsFirst(t *testing.T) {
	row := importer.ParsedRow{Date: "bad", AmountInput: "$1", ParseErr: errors.New("invalid date")}

	blocker := rowSaveBlocker(row, "Groceries")

	if !strings.Contains(blocker, "invalid date") {
		t.Errorf("a row parse error should be reported ahead of the amount: %q", blocker)
	}
}

func TestRowSaveBlockerRequiresACategory(t *testing.T) {
	row := importer.ParsedRow{Date: "2026-08-01", Type: "expense", AmountInput: "45.67"}

	if blocker := rowSaveBlocker(row, ""); !strings.Contains(blocker, "category") {
		t.Errorf("an empty category should block the save, got %q", blocker)
	}
}

func TestRowSaveBlockerAllowsAValidRow(t *testing.T) {
	row := importer.ParsedRow{Date: "2026-08-01", Type: "expense", AmountInput: "45.67"}

	if blocker := rowSaveBlocker(row, "Groceries"); blocker != "" {
		t.Errorf("a valid row should not be blocked, got %q", blocker)
	}
}
