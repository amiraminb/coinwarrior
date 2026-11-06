package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/daterange"
	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
)

// Recurring costs are grouped by currency alongside the saved budgets. The
// schedule supplies the planned amount; only recorded recurring expenses count
// as spent, including transactions whose rule has since been deleted.
func recurringBudgetSummariesForMonth(budgets []BudgetSummary, transactions []model.Transaction, rules []model.RecurringRule, month time.Time) ([]BudgetSummary, error) {
	start, end := daterange.MonthBounds(month)
	byCurrency := make(map[string]*BudgetSummary)
	getSummary := func(currency string) *BudgetSummary {
		currency = money.NormalizeCurrency(currency)
		if summary, ok := byCurrency[currency]; ok {
			return summary
		}
		summary := &BudgetSummary{
			Budget:      model.Budget{Month: daterange.FormatMonth(month), Currency: currency},
			Recurring:   true,
			PeriodStart: start,
			PeriodEnd:   end,
		}
		byCurrency[currency] = summary
		return summary
	}
	for _, budget := range budgets {
		getSummary(budget.Budget.Currency)
	}
	for _, rule := range rules {
		if rule.Type != model.TransactionTypeExpense {
			continue
		}
		scheduled, err := recurringExpenseScheduledInMonth(rule, month)
		if err != nil {
			return nil, err
		}
		if scheduled {
			getSummary(rule.Currency).Budget.AmountMinor += rule.AmountMinor
		}
	}
	for _, tx := range transactions {
		if tx.Type != model.TransactionTypeExpense || tx.Source != model.TransactionSourceRecurring {
			continue
		}
		inRange, err := daterange.Contains(tx.Date, start, end)
		if err != nil {
			return nil, err
		}
		if inRange {
			getSummary(tx.Currency).SpentMinor += tx.AmountMinor
		}
	}

	summaries := make([]BudgetSummary, 0, len(byCurrency))
	for _, summary := range byCurrency {
		summary.LeftMinor = summary.Budget.AmountMinor - summary.SpentMinor
		summaries = append(summaries, *summary)
	}
	return summaries, nil
}

func recurringExpenseScheduledInMonth(rule model.RecurringRule, month time.Time) (bool, error) {
	start, err := time.Parse("2006-01-02", rule.StartDate)
	if err != nil {
		return false, fmt.Errorf("rule %s: invalid start date: %s", rule.ID, rule.StartDate)
	}
	if rule.DayOfMonth < recurringDayOfMonthMin || rule.DayOfMonth > recurringDayOfMonthMax {
		return false, fmt.Errorf("rule %s: day of month must be between %d and %d", rule.ID, recurringDayOfMonthMin, recurringDayOfMonthMax)
	}
	// Compare date-only values in UTC, independently of the report's time zone.
	occurrence := time.Date(month.Year(), month.Month(), rule.DayOfMonth, 0, 0, 0, 0, time.UTC)
	var end time.Time
	if strings.TrimSpace(rule.EndDate) != "" {
		end, err = time.Parse("2006-01-02", rule.EndDate)
		if err != nil {
			return false, fmt.Errorf("rule %s: invalid end date: %s", rule.ID, rule.EndDate)
		}
	}
	return !occurrence.Before(start) && (end.IsZero() || !occurrence.After(end)), nil
}
