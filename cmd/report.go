package cmd

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"
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
	reportWorseStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	reportBetterStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	reportMutedStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show reports",
	Long: `Show reports.

Subcommands:
  account                           account balances and totals
  overview [category] <range>       category breakdown, budget, and totals for a range
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
	Use:   "overview [category] <range>",
	Short: "Show the category breakdown, budget, and totals for a range",
	Long: `Show the category breakdown, budget, and totals for a range.

Prints per-category totals, an income/expense summary, and, when the range is
exactly one calendar month, that month's budget. A range spanning more than one
month also gets a per-month income/expense bar chart, with partly covered months
labelled by the days they cover, and the same breakdown repeated per month.

When a category is given, every section covers only that category. It is matched
case-insensitively against the saved categories, and transfers are excluded.

Supported ranges: today, yesterday, week, lastweek, month, lastmonth, year, lastyear, or YYYY-MM-DD..YYYY-MM-DD.`,
	Example: `  coinw report overview month
  coinw report overview Groceries year
  coinw report overview 2026-04-01..2026-04-30`,
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: completeTransactionsArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		start, end, err := daterange.Resolve(args[len(args)-1], time.Now())
		if err != nil {
			if len(args) == 1 {
				return fmt.Errorf("%w (usage: coinw report overview [category] <range>)", err)
			}
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
		allTransactions := transactions
		transactions = filterByCategory(transactions, category)

		heading := fmt.Sprintf("overview %s..%s", start.Format("2006-01-02"), end.Format("2006-01-02"))
		if category != "" {
			heading = fmt.Sprintf("overview %s %s..%s", category, start.Format("2006-01-02"), end.Format("2006-01-02"))
		}
		fmt.Println(tui.HeaderStyle.Render(heading))
		fmt.Println()
		if err := printCategorySection(transactions, allTransactions, start, end, time.Now(), category != ""); err != nil {
			return err
		}
		printMonthlyBarsSection(transactions, start, end)
		printPerMonthSections(transactions, allTransactions, start, end)

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

func printCategorySection(transactions, shareBase []model.Transaction, start, end time.Time, now time.Time, filtered bool) error {
	fmt.Println(reportSectionStyle.Render("Range Categories"))
	fmt.Println()
	// A budget covers the whole month with no category dimension, so under a
	// category filter it can only be omitted, not narrowed.
	if !filtered {
		if err := printMonthlyBudgetSection(start, end, now); err != nil {
			return err
		}
	}

	if !printCategoryTotals(transactions, shareBase, start, end, "Category Totals (Range)") {
		fmt.Println("  no transactions in range")
		return nil
	}

	fmt.Println()
	return nil
}

// Filtering once here means every section below inherits the category, and
// transfers drop out because they never carry a non-transfer category.
func filterByCategory(transactions []model.Transaction, category string) []model.Transaction {
	if category == "" {
		return transactions
	}

	filtered := make([]model.Transaction, 0, len(transactions))
	for _, tx := range transactions {
		if tx.Type == model.TransactionTypeTransfer {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(tx.Category), category) {
			filtered = append(filtered, tx)
		}
	}

	return filtered
}

// categoryKey identifies one row of the totals table across months, so a month
// can be compared against the one before it.
type categoryKey struct {
	category string
	currency string
}

func printCategoryTotals(transactions, shareBase []model.Transaction, start, end time.Time, heading string) bool {
	_, ok := printCategoryTotalsWithBaseline(transactions, shareBase, start, end, heading, nil)
	return ok
}

// shareBase is the unfiltered set, so % EXP stays each category's share of all
// spending rather than collapsing to 100% under a category filter.
func printCategoryTotalsWithBaseline(transactions, shareBase []model.Transaction, start, end time.Time, heading string, baseline map[categoryKey]int64) (map[categoryKey]int64, bool) {
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
		return nil, false
	}

	fmt.Println(reportSubSectionStyle.Render(heading))
	fmt.Println()
	shareDenominator := expenseByCurrency(shareBase, start, end)
	totals := make(map[categoryKey]int64)
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
			if totalExpense := shareDenominator[currency]; totalExpense > 0 {
				expenseShare = formatPercent(report.expenseByCurrency[currency], totalExpense)
			}

			total := report.totalsByCurrency[currency]
			key := categoryKey{category: report.name, currency: currency}
			totals[key] = total

			row := table.Row{displayCategory, currency, money.Format(total), strconv.Itoa(len(report.items)), expenseShare}
			if baseline != nil {
				row = append(row, formatMonthDelta(total, key, baseline))
			}
			totalRows = append(totalRows, row)
		}
	}

	columns := []table.Column{
		{Title: "CATEGORY", Width: 20},
		{Title: "CUR", Width: 5},
		{Title: "TOTAL", Width: 14},
		{Title: "TXNS", Width: 6},
		{Title: "% EXP", Width: 8},
	}
	if baseline != nil {
		columns = append(columns, table.Column{Title: "VS PREV", Width: widestCell(totalRows, len(columns)) + 2})
	}
	tui.RenderTable(columns, totalRows)

	printIncomeExpenseSummary(currencyIncome, currencyExpense)

	return totals, true
}

func expenseByCurrency(transactions []model.Transaction, start, end time.Time) map[string]int64 {
	totals := make(map[string]int64)
	for _, tx := range transactions {
		if tx.Type != model.TransactionTypeExpense {
			continue
		}
		inRange, err := daterange.Contains(tx.Date, start, end)
		if err != nil || !inRange {
			continue
		}
		totals[tx.Currency] += tx.AmountMinor
	}
	return totals
}

func widestCell(rows []table.Row, column int) int {
	widest := 0
	for _, row := range rows {
		if column < len(row) {
			widest = max(widest, len(row[column]))
		}
	}
	return widest
}

// Colour reflects whether the month moved for the worse, not whether the signed
// number rose: spending more is red even though the total falls.
func formatMonthDelta(total int64, key categoryKey, baseline map[categoryKey]int64) string {
	previous, ok := baseline[key]
	if !ok {
		return reportMutedStyle.Render("new")
	}

	change := total - previous
	if change == 0 {
		return reportMutedStyle.Render("=")
	}

	if change < 0 {
		return reportWorseStyle.Render("▲ " + money.Format(-change))
	}
	return reportBetterStyle.Render("▼ " + money.Format(change))
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
