package cmd

import (
	"fmt"
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
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show reports",
	Long: `Show reports.

Subcommands:
  account              account balances and totals
  budget <range>       budget and category breakdown for a range
  transactions <range> transactions in a range`,
}

var reportBudgetCmd = &cobra.Command{
	Use:   "budget <range>",
	Short: "Show budget and category breakdown for a range",
	Long: `Show budget and category breakdown for a range.

Supported ranges: today, yesterday, week, lastweek, month, lastmonth, year, lastyear, or YYYY-MM-DD..YYYY-MM-DD.`,
	Example: `  coinw report budget month
  coinw report budget 2026-04-01..2026-04-30`,
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

		fmt.Println(tui.HeaderStyle.Render(fmt.Sprintf("report %s..%s", start.Format("2006-01-02"), end.Format("2006-01-02"))))
		fmt.Println()
		if err := printCategorySection(transactions, start, end, time.Now()); err != nil {
			return err
		}

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
		fmt.Println("  no transactions in range")
		return nil
	}

	fmt.Println(reportSubSectionStyle.Render("Category Totals (Range)"))
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

	summaryRows := make([]table.Row, 0)
	allCurrenciesMap := make(map[string]bool)
	for c := range currencyIncome {
		allCurrenciesMap[c] = true
	}
	for c := range currencyExpense {
		allCurrenciesMap[c] = true
	}
	allCurrencies := make([]string, 0, len(allCurrenciesMap))
	for c := range allCurrenciesMap {
		allCurrencies = append(allCurrencies, c)
	}
	sort.Strings(allCurrencies)

	for _, c := range allCurrencies {
		income := currencyIncome[c]
		expense := currencyExpense[c]
		net := income - expense
		summaryRows = append(summaryRows, table.Row{c, money.Format(income), money.Format(expense), money.Format(net)})
	}

	if len(summaryRows) > 0 {
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
			summaryRows,
		)
	}

	fmt.Println()
	return nil
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
	reportCmd.AddCommand(reportBudgetCmd)
	reportCmd.AddCommand(reportAccountCmd)
	reportCmd.AddCommand(reportTransactionsCmd)
	rootCmd.AddCommand(reportCmd)
}
