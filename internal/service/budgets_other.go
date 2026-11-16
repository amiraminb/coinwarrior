package service

import (
	"strings"

	"github.com/amiraminb/coinwarrior/internal/daterange"
	"github.com/amiraminb/coinwarrior/internal/model"
)

// Other spending reserves recurring costs and explicit category limits. Its
// actual spending includes only non-recurring expenses in unbudgeted categories.
func otherBudgetSummaries(summaries []BudgetSummary, transactions []model.Transaction) ([]BudgetSummary, error) {
	var other []BudgetSummary
	for _, overall := range summaries {
		if overall.Recurring || overall.Other || overall.Budget.Category != "" {
			continue
		}
		remainder := BudgetSummary{
			Budget: model.Budget{
				Month:         overall.Budget.Month,
				Currency:      overall.Budget.Currency,
				AmountMinor:   overall.Budget.AmountMinor,
				RolloverMinor: overall.Budget.RolloverMinor,
			},
			Other:       true,
			PeriodStart: overall.PeriodStart,
			PeriodEnd:   overall.PeriodEnd,
		}
		budgetedCategories := make(map[string]bool)
		for _, allocation := range summaries {
			if allocation.Other || (!allocation.Recurring && allocation.Budget.Category == "") ||
				allocation.Budget.Month != overall.Budget.Month ||
				!strings.EqualFold(allocation.Budget.Currency, overall.Budget.Currency) {
				continue
			}
			remainder.Budget.AmountMinor -= allocation.Budget.AmountMinor
			remainder.Budget.RolloverMinor -= allocation.Budget.RolloverMinor
			if !allocation.Recurring {
				budgetedCategories[strings.ToLower(allocation.Budget.Category)] = true
			}
		}
		// Filter the union of recurring and category-budgeted expenses rather
		// than subtracting both totals, which can contain the same transaction.
		for _, tx := range transactions {
			if tx.Type != model.TransactionTypeExpense || tx.Source == model.TransactionSourceRecurring ||
				!strings.EqualFold(tx.Currency, overall.Budget.Currency) ||
				budgetedCategories[strings.ToLower(strings.TrimSpace(tx.Category))] {
				continue
			}
			inRange, err := daterange.Contains(tx.Date, overall.PeriodStart, overall.PeriodEnd)
			if err != nil {
				return nil, err
			}
			if inRange {
				remainder.SpentMinor += tx.AmountMinor
			}
		}
		remainder.LeftMinor = remainder.Budget.AmountMinor + remainder.Budget.RolloverMinor - remainder.SpentMinor
		other = append(other, remainder)
	}
	return other, nil
}
