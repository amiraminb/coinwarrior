package cmd

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/amiraminb/coinwarrior/internal/daterange"
	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	reportSectionStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111"))
	reportSubSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("150"))
	reportMonthStyle      = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("117"))
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show reports",
	Long: `Show reports.

Subcommands:
  account                           account balances and totals
  overview <range>                  category breakdown, budget, and totals for a range
  transactions [category] <range>   transactions in a range, optionally for one category`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unknown report subcommand '%s' (see 'coinw report --help')", args[0])
		}
		return cmd.Help()
	},
}

var reportOverviewCmd = &cobra.Command{
	Use:   "overview <range>",
	Short: "Show the category breakdown, budget, and totals for a range",
	Long: `Show the category breakdown, budget, and totals for a range.

Prints per-category totals, an income/expense summary, and, when the range is
exactly one calendar month, that month's budget. A range spanning more than one
month also gets a per-month income/expense bar chart, with partly covered months
labelled by the days they cover.

Supported ranges: today, yesterday, week, lastweek, month, lastmonth, year, lastyear, or YYYY-MM-DD..YYYY-MM-DD.`,
	Example: `  coinw report overview month
  coinw report overview 2026-04-01..2026-04-30`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		start, end, err := daterange.Resolve(args[0], time.Now())
		if err != nil {
			return err
		}

		transactions, err := repo.LoadTransactions()
		if err != nil {
			return err
		}

		fmt.Println(tui.HeaderStyle.Render(fmt.Sprintf("overview %s..%s", start.Format("2006-01-02"), end.Format("2006-01-02"))))
		fmt.Println()
		if err := printCategorySection(transactions, start, end, time.Now()); err != nil {
			return err
		}
		printMonthlyBarsSection(transactions, start, end)
		printPerMonthSections(transactions, start, end)

		return nil
	},
}

var reportAccountCmd = &cobra.Command{
	Use:     "account",
	Short:   "Show account balances and totals",
	Example: `  coinw report account`,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAccountReport()
	},
}

func runAccountReport() error {
	accounts, err := repo.LoadAccounts()
	if err != nil {
		return err
	}

	fmt.Println(tui.HeaderStyle.Render("account report"))
	fmt.Println()
	printAccountBalancesReport(accounts)
	fmt.Println()
	printTotalBalancesReport(accounts)
	fmt.Println()

	return nil
}

func printAccountBalancesReport(accounts []model.Account) {
	fmt.Println(reportSubSectionStyle.Render("Account Balances"))
	if len(accounts) == 0 {
		fmt.Println("  no accounts")
		return
	}

	items := make([]model.Account, len(accounts))
	copy(items, accounts)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	rows := make([]table.Row, 0, len(items))
	for _, account := range items {
		rows = append(rows, table.Row{account.Name, account.Currency, money.Format(account.BalanceMinor)})
	}

	tui.RenderTable(
		[]table.Column{
			{Title: "ACCOUNT", Width: 24},
			{Title: "CUR", Width: 5},
			{Title: "BALANCE", Width: 14},
		},
		rows,
	)
}

func printTotalBalancesReport(accounts []model.Account) {
	fmt.Println(reportSubSectionStyle.Render("Total Balances"))
	if len(accounts) == 0 {
		fmt.Println("  no balances")
		return
	}

	totals := make(map[string]int64)
	for _, account := range accounts {
		totals[account.Currency] += account.BalanceMinor
	}

	currencies := make([]string, 0, len(totals))
	for currency := range totals {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)

	rows := make([]table.Row, 0, len(currencies))
	for _, currency := range currencies {
		rows = append(rows, table.Row{currency, money.Format(totals[currency])})
	}

	tui.RenderTable(
		[]table.Column{
			{Title: "CUR", Width: 5},
			{Title: "TOTAL", Width: 14},
		},
		rows,
	)
}

func printCategorySection(transactions []model.Transaction, start, end time.Time, now time.Time) error {
	fmt.Println(reportSectionStyle.Render("Range Categories"))
	fmt.Println()
	if err := printMonthlyBudgetSection(start, end, now); err != nil {
		return err
	}

	if !printCategoryTotals(transactions, start, end, "Category Totals (Range)") {
		fmt.Println("  no transactions in range")
		return nil
	}

	fmt.Println()
	return nil
}

func printCategoryTotals(transactions []model.Transaction, start, end time.Time, heading string) bool {
	byCategory := make(map[string][]model.Transaction)
	for _, tx := range transactions {
		if tx.Type == model.TransactionTypeTransfer {
			continue
		}
		inRange, err := daterange.Contains(tx.Date, start, end)
		if err != nil || !inRange {
			continue
		}
		byCategory[tx.Category] = append(byCategory[tx.Category], tx)
	}

	if len(byCategory) == 0 {
		return false
	}

	fmt.Println(reportSubSectionStyle.Render(heading))
	fmt.Println()
	totalRows := make([]table.Row, 0)
	currencyIncome := make(map[string]int64)
	currencyExpense := make(map[string]int64)
	categoryReports := make([]categoryReport, 0, len(byCategory))
	for category, items := range byCategory {
		report := categoryReport{
			name:              category,
			items:             items,
			totalsByCurrency:  make(map[string]int64),
			expenseByCurrency: make(map[string]int64),
		}

		for _, tx := range items {
			if tx.Type == model.TransactionTypeExpense {
				currencyExpense[tx.Currency] += tx.AmountMinor
				report.expenseByCurrency[tx.Currency] += tx.AmountMinor
				report.totalExpenseMinor += tx.AmountMinor
			} else if tx.Type == model.TransactionTypeIncome {
				currencyIncome[tx.Currency] += tx.AmountMinor
			}

			delta := tx.AmountMinor
			if tx.Type == model.TransactionTypeExpense {
				delta = -tx.AmountMinor
			}
			report.totalsByCurrency[tx.Currency] += delta
		}
		categoryReports = append(categoryReports, report)
	}

	sort.Slice(categoryReports, func(i, j int) bool {
		if categoryReports[i].totalExpenseMinor == categoryReports[j].totalExpenseMinor {
			return categoryReports[i].name < categoryReports[j].name
		}
		return categoryReports[i].totalExpenseMinor > categoryReports[j].totalExpenseMinor
	})

	for _, report := range categoryReports {
		currencies := make([]string, 0, len(report.totalsByCurrency))
		for currency := range report.totalsByCurrency {
			currencies = append(currencies, currency)
		}
		sort.Strings(currencies)

		displayCategory := report.name
		if displayCategory == "" {
			displayCategory = "(no category)"
		}

		for _, currency := range currencies {
			expenseShare := "-"
			if totalExpense := currencyExpense[currency]; totalExpense > 0 {
				expenseShare = formatPercent(report.expenseByCurrency[currency], totalExpense)
			}
			totalRows = append(totalRows, table.Row{displayCategory, currency, money.Format(report.totalsByCurrency[currency]), strconv.Itoa(len(report.items)), expenseShare})
		}
	}

	tui.RenderTable(
		[]table.Column{
			{Title: "CATEGORY", Width: 20},
			{Title: "CUR", Width: 5},
			{Title: "TOTAL", Width: 14},
			{Title: "TXNS", Width: 6},
			{Title: "% EXP", Width: 8},
		},
		totalRows,
	)

	printIncomeExpenseSummary(currencyIncome, currencyExpense)

	return true
}

func printIncomeExpenseSummary(income, expense map[string]int64) {
	currencies := make(map[string]bool, len(income)+len(expense))
	for currency := range income {
		currencies[currency] = true
	}
	for currency := range expense {
		currencies[currency] = true
	}
	if len(currencies) == 0 {
		return
	}

	ordered := slices.Sorted(maps.Keys(currencies))
	rows := make([]table.Row, 0, len(ordered))
	for _, currency := range ordered {
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

func printMonthlyBudgetSection(start, end, now time.Time) error {
	monthLabel, ok := budgetMonthForRange(start, end)
	if !ok {
		return nil
	}

	fmt.Println(reportSubSectionStyle.Render("Monthly Budget"))
	fmt.Println()

	summaries, err := svc.GetMonthlyBudgetSummaries(monthLabel, now)
	if err != nil {
		return err
	}
	if len(summaries) == 0 {
		fmt.Printf("no budget set for %s\n", monthLabel)
		fmt.Println()
		return nil
	}

	rows := make([]table.Row, 0, len(summaries))
	for _, summary := range summaries {
		rows = append(rows, table.Row{
			summary.Budget.Currency,
			money.Format(summary.Budget.AmountMinor),
			money.Format(summary.Budget.RolloverMinor),
			money.Format(summary.SpentMinor),
			money.Format(summary.LeftMinor),
		})
	}

	tui.RenderTable(
		[]table.Column{
			{Title: "CUR", Width: 5},
			{Title: "BUDGET", Width: 14},
			{Title: "ROLL", Width: 14},
			{Title: "SPENT", Width: 14},
			{Title: "LEFT", Width: 14},
		},
		rows,
	)
	fmt.Println()

	return nil
}

func budgetMonthForRange(start, end time.Time) (string, bool) {
	if start.Year() != end.Year() || start.Month() != end.Month() {
		return "", false
	}
	if start.Day() != 1 {
		return "", false
	}
	monthEnd := time.Date(start.Year(), start.Month()+1, 0, 0, 0, 0, 0, start.Location())
	if end.Year() != monthEnd.Year() || end.Month() != monthEnd.Month() || end.Day() != monthEnd.Day() {
		return "", false
	}

	return daterange.FormatMonth(start), true
}

type categoryReport struct {
	name              string
	items             []model.Transaction
	totalsByCurrency  map[string]int64
	expenseByCurrency map[string]int64
	totalExpenseMinor int64
}

func formatPercent(part, total int64) string {
	if total <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", (float64(part)*100)/float64(total))
}

func init() {
	reportCmd.AddCommand(reportOverviewCmd)
	reportCmd.AddCommand(reportAccountCmd)
	reportCmd.AddCommand(reportTransactionsCmd)
	rootCmd.AddCommand(reportCmd)
}
