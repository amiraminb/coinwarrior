package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/daterange"
	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
)

var reportTransactionsCmd = &cobra.Command{
	Use:   "transactions [category] <range>",
	Short: "List transactions in a range, optionally for one category",
	Long: `List transactions in a range, optionally filtered to a single category.

Supported ranges: today, yesterday, week, lastweek, month, lastmonth, year, lastyear, or YYYY-MM-DD..YYYY-MM-DD.

When a category is given it is matched case-insensitively against the saved
categories, and an income/expense summary for that category is printed below
the table.`,
	Example: `  coinw report transactions month
  coinw report transactions yesterday
  coinw report transactions Groceries month
  coinw report transactions groceries 2026-04-01..2026-04-30`,
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: completeTransactionsArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		rangeArg := args[len(args)-1]
		start, end, err := daterange.Resolve(rangeArg, time.Now())
		if err != nil {
			return err
		}

		category := ""
		if len(args) == 2 {
			category, err = svc.ResolveCategory(args[0])
			if err != nil {
				return err
			}
		}

		transactions, err := repo.LoadTransactions()
		if err != nil {
			return err
		}

		items := make([]model.Transaction, 0, len(transactions))
		for _, tx := range transactions {
			if category != "" && !strings.EqualFold(tx.Category, category) {
				continue
			}
			inRange, err := daterange.Contains(tx.Date, start, end)
			if err != nil {
				return fmt.Errorf("invalid transaction date '%s' for %s", tx.Date, tx.ID)
			}
			if inRange {
				items = append(items, tx)
			}
		}

		if category != "" {
			fmt.Println(tui.HeaderStyle.Render(fmt.Sprintf("%s %s..%s", category, start.Format("2006-01-02"), end.Format("2006-01-02"))))
			fmt.Println()
		}

		if len(items) == 0 {
			fmt.Println("no transactions")
			return nil
		}

		tui.SortTransactionsByDateDesc(items)

		columns := []table.Column{
			{Title: "ID", Width: 24},
			{Title: "DATE", Width: 10},
			{Title: "TYPE", Width: 8},
			{Title: "AMOUNT", Width: 12},
			{Title: "CUR", Width: 5},
			{Title: "CATEGORY", Width: 16},
			{Title: "ACCOUNT", Width: 16},
			{Title: "NOTE", Width: 20},
		}

		rows := make([]table.Row, 0, len(items))
		for _, tx := range items {
			account := tx.Account
			if tx.Type == model.TransactionTypeTransfer {
				account = tx.Account + " -> " + tx.ToAccount
			}
			rows = append(rows, table.Row{
				tx.ID,
				tx.Date,
				tx.Type,
				money.Format(tx.AmountMinor),
				tx.Currency,
				tx.Category,
				account,
				tx.Note,
			})
		}

		tui.RenderTable(columns, rows)

		if category != "" {
			printTransactionsSummary(items)
		}

		return nil
	},
}

func completeTransactionsArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		categories, err := svc.LoadCategories()
		if err != nil {
			return daterange.Names(), cobra.ShellCompDirectiveNoFileComp
		}
		return append(categories, daterange.Names()...), cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) == 1 {
		return daterange.Names(), cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func printTransactionsSummary(items []model.Transaction) {
	income := make(map[string]int64)
	expense := make(map[string]int64)
	for _, tx := range items {
		switch tx.Type {
		case model.TransactionTypeIncome:
			income[tx.Currency] += tx.AmountMinor
		case model.TransactionTypeExpense:
			expense[tx.Currency] += tx.AmountMinor
		}
	}

	currencies := make([]string, 0, len(income)+len(expense))
	for currency := range income {
		currencies = append(currencies, currency)
	}
	for currency := range expense {
		if _, ok := income[currency]; !ok {
			currencies = append(currencies, currency)
		}
	}
	if len(currencies) == 0 {
		return
	}
	sort.Strings(currencies)

	rows := make([]table.Row, 0, len(currencies))
	for _, currency := range currencies {
		rows = append(rows, table.Row{
			currency,
			money.Format(income[currency]),
			money.Format(expense[currency]),
			money.Format(income[currency] - expense[currency]),
		})
	}

	fmt.Println()
	fmt.Println(reportSubSectionStyle.Render("Income / Expense Summary"))
	fmt.Println()
	tui.RenderTable(
		[]table.Column{
			{Title: "CUR", Width: 5},
			{Title: "INCOME", Width: 14},
			{Title: "EXPENSE", Width: 14},
			{Title: "NET", Width: 14},
		},
		rows,
	)
}
