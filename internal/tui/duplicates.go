package tui

import (
	"fmt"
	"strings"

	"github.com/amiraminb/coinwarrior/internal/model"
)

func confirmNotDuplicate(txType, amountInput, currency, dateValue, category string) (bool, error) {
	matches, err := Svc.FindPossibleDuplicates(txType, amountInput, dateValue, category)
	if err != nil {
		return false, err
	}
	if len(matches) == 0 {
		return true, nil
	}

	return RunConfirmPrompt(duplicateWarning(matches, amountInput, currency, dateValue, category))
}

func duplicateWarning(matches []model.Transaction, amountInput, currency, dateValue, category string) string {
	subject := "transaction already matches"
	if len(matches) > 1 {
		subject = "transactions already match"
	}

	var b strings.Builder
	b.WriteString(warnStyle.Render(fmt.Sprintf("This might be a duplicate: %d existing %s this date, category, amount, and type.", len(matches), subject)))
	b.WriteString("\n\n")
	for _, tx := range matches {
		b.WriteString(mutedStyle.Render("  " + FormatEditableTransaction(tx)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Add %s %s to %s on %s anyway?", strings.TrimSpace(amountInput), currency, category, dateValue))

	return b.String()
}
