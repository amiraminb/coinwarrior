package cmd

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/daterange"
	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/charmbracelet/bubbles/table"
)

const monthlyBarWidth = 24

type monthlyTotals struct {
	label        string
	incomeMinor  int64
	expenseMinor int64
}

type monthSpan struct {
	label string
	key   string
	start time.Time
	end   time.Time
}

// A partly covered month is labelled with its covered days, so a short bar is
// not misread as low activity.
func monthSpansInRange(start, end time.Time) []monthSpan {
	var spans []monthSpan
	for cursor := start; !cursor.After(end); {
		monthStart, monthEnd := daterange.MonthBounds(cursor)

		coveredStart := monthStart
		if start.After(coveredStart) {
			coveredStart = start
		}
		coveredEnd := monthEnd
		if end.Before(coveredEnd) {
			coveredEnd = end
		}

		label := daterange.FormatMonth(monthStart)
		if !coveredStart.Equal(monthStart) || !coveredEnd.Equal(monthEnd) {
			label += fmt.Sprintf(" (%02d-%02d)", coveredStart.Day(), coveredEnd.Day())
		}
		spans = append(spans, monthSpan{
			label: label,
			key:   daterange.FormatMonth(monthStart),
			start: coveredStart,
			end:   coveredEnd,
		})

		cursor = monthEnd.AddDate(0, 0, 1)
	}

	return spans
}

func monthsInRange(start, end time.Time) []string {
	spans := monthSpansInRange(start, end)
	labels := make([]string, 0, len(spans))
	for _, span := range spans {
		labels = append(labels, span.label)
	}
	return labels
}

// Transfers are excluded, matching every other section of the overview: moving
// money between accounts is neither earned nor spent.
func monthlyBreakdown(transactions []model.Transaction, start, end time.Time) map[string][]monthlyTotals {
	labels := monthsInRange(start, end)
	if len(labels) == 0 {
		return nil
	}

	index := make(map[string]int, len(labels))
	for i, label := range labels {
		index[strings.SplitN(label, " ", 2)[0]] = i
	}

	byCurrency := make(map[string][]monthlyTotals)
	for _, tx := range transactions {
		if tx.Type != model.TransactionTypeIncome && tx.Type != model.TransactionTypeExpense {
			continue
		}
		inRange, err := daterange.Contains(tx.Date, start, end)
		if err != nil || !inRange {
			continue
		}
		i, ok := index[monthKeyOf(tx.Date)]
		if !ok {
			continue
		}

		rows, seen := byCurrency[tx.Currency]
		if !seen {
			rows = make([]monthlyTotals, len(labels))
			for j, label := range labels {
				rows[j].label = label
			}
		}
		switch tx.Type {
		case model.TransactionTypeIncome:
			rows[i].incomeMinor += tx.AmountMinor
		case model.TransactionTypeExpense:
			rows[i].expenseMinor += tx.AmountMinor
		}
		byCurrency[tx.Currency] = rows
	}

	return byCurrency
}

func monthKeyOf(date string) string {
	if len(date) < 7 {
		return ""
	}
	return date[:7]
}

// printMonthlyBarsSection is skipped for a single calendar month, where the
// bars would be one row and the summary above already states those totals.
func printMonthlyBarsSection(transactions []model.Transaction, start, end time.Time) {
	if len(monthsInRange(start, end)) < 2 {
		return
	}

	byCurrency := monthlyBreakdown(transactions, start, end)
	if len(byCurrency) == 0 {
		return
	}

	for _, currency := range slices.Sorted(maps.Keys(byCurrency)) {
		rows := byCurrency[currency]

		peak := int64(0)
		for _, row := range rows {
			peak = max(peak, row.incomeMinor, row.expenseMinor)
		}

		labelWidth := 0
		for _, row := range rows {
			labelWidth = max(labelWidth, len(row.label))
		}

		tableRows := make([]table.Row, 0, len(rows))
		for _, row := range rows {
			tableRows = append(tableRows, table.Row{
				row.label,
				barCell(row.incomeMinor, peak),
				barCell(row.expenseMinor, peak),
			})
		}

		fmt.Println()
		fmt.Println(reportSubSectionStyle.Render("Monthly Income / Expense (" + currency + ")"))
		fmt.Println()
		tui.RenderTable(
			[]table.Column{
				{Title: "MONTH", Width: labelWidth + 2},
				{Title: "INCOME", Width: monthlyBarWidth + 14},
				{Title: "EXPENSE", Width: monthlyBarWidth + 14},
			},
			tableRows,
		)
	}
}

// Skipped for a single month, where these totals would restate the range
// section printed above.
func printPerMonthSections(transactions []model.Transaction, start, end time.Time) {
	spans := monthSpansInRange(start, end)
	if len(spans) < 2 || !hasReportableTransaction(transactions, start, end) {
		return
	}

	fmt.Println()
	fmt.Println(reportSectionStyle.Render("Per-Month Breakdown"))

	for _, span := range spans {
		fmt.Println()
		fmt.Println(reportMonthStyle.Render(span.label))
		fmt.Println()
		if !printCategoryTotals(transactions, span.start, span.end, "Category Totals") {
			fmt.Println("  no transactions")
		}
	}
}

func hasReportableTransaction(transactions []model.Transaction, start, end time.Time) bool {
	return slices.ContainsFunc(transactions, func(tx model.Transaction) bool {
		if tx.Type == model.TransactionTypeTransfer {
			return false
		}
		inRange, err := daterange.Contains(tx.Date, start, end)
		return err == nil && inRange
	})
}

func barCell(amountMinor, peakMinor int64) string {
	amount := money.Format(amountMinor)
	if amountMinor <= 0 || peakMinor <= 0 {
		return amount
	}
	return bar(amountMinor, peakMinor) + " " + amount
}

// bar scales to the largest value across both columns so income and expense
// stay visually comparable, and never renders empty for a non-zero amount.
func bar(amountMinor, peakMinor int64) string {
	if amountMinor <= 0 || peakMinor <= 0 {
		return ""
	}

	filled := int((float64(amountMinor) / float64(peakMinor)) * monthlyBarWidth)
	return strings.Repeat("█", max(filled, 1))
}
