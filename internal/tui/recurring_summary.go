package tui

import (
	"fmt"
	"maps"
	"slices"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
	"github.com/charmbracelet/bubbles/table"
)

func printRecurringSummary(rules []model.RecurringRule) {
	type monthlyTotals struct {
		income  int64
		expense int64
	}
	byCurrency := make(map[string]monthlyTotals)
	for _, rule := range rules {
		totals := byCurrency[rule.Currency]
		switch rule.Type {
		case model.TransactionTypeIncome:
			totals.income += rule.AmountMinor
		case model.TransactionTypeExpense:
			totals.expense += rule.AmountMinor
		default:
			continue
		}
		byCurrency[rule.Currency] = totals
	}
	if len(byCurrency) == 0 {
		return
	}

	summaryRows := make([]table.Row, 0, len(byCurrency))
	futureRows := make([]table.Row, 0, len(byCurrency)*3)
	for _, currency := range slices.Sorted(maps.Keys(byCurrency)) {
		totals := byCurrency[currency]
		net := totals.income - totals.expense
		summaryRows = append(summaryRows, table.Row{
			currency, money.Format(totals.income), money.Format(totals.expense), money.Format(net),
		})
		for _, item := range []struct {
			label  string
			amount int64
		}{
			{"Income", totals.income}, {"Expense", totals.expense}, {"Net", net},
		} {
			row := table.Row{currency, item.label, money.Format(item.amount)}
			for _, years := range money.FutureValueHorizons() {
				row = append(row, money.Format(money.FutureValueMinor(item.amount, years)))
			}
			futureRows = append(futureRows, row)
		}
	}

	fmt.Println()
	fmt.Println(HeaderStyle.Render("Monthly Income / Expense Summary (Listed Rules)"))
	fmt.Println()
	RenderTable([]table.Column{
		{Title: "CUR", Width: 5},
		{Title: "INCOME", Width: 14},
		{Title: "EXPENSE", Width: 14},
		{Title: "NET", Width: 14},
	}, summaryRows)

	fmt.Println()
	fmt.Println(HeaderStyle.Render(fmt.Sprintf("Future Value of One Month's Totals (%d%% Annual Interest)", money.FutureValueInterestPercent)))
	fmt.Println()
	columns := []table.Column{
		{Title: "CUR", Width: 5},
		{Title: "TYPE", Width: 8},
		{Title: "AMOUNT", Width: 14},
	}
	for _, years := range money.FutureValueHorizons() {
		columns = append(columns, table.Column{Title: fmt.Sprintf("%d YEARS", years), Width: 14})
	}
	RenderTable(columns, futureRows)
}
