package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	reportSectionStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111"))
	reportSubSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("150"))
	reportShowDetails     bool
)

var reportCmd = &cobra.Command{
	Use:   "report <range|account>",
	Short: "Show reports",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.EqualFold(args[0], "account") {
			return runAccountReport()
		}

		start, end, err := coininternal.ResolveDateRange(args[0], time.Now())
		if err != nil {
			return err
		}

		transactionsPath, err := coininternal.FilePath(coininternal.TransactionsFileName)
		if err != nil {
			return err
		}
		transactionsFile, err := coininternal.LoadTransactions(transactionsPath)
		if err != nil {
			return err
		}

		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("250"))
		fmt.Println(headerStyle.Render(fmt.Sprintf("report %s..%s", start.Format("2006-01-02"), end.Format("2006-01-02"))))
		fmt.Println()
		printCategorySection(transactionsFile.Transactions, start, end, reportShowDetails)

		return nil
	},
}

func runAccountReport() error {
	accountsPath, err := coininternal.FilePath(coininternal.AccountsFileName)
	if err != nil {
		return err
	}

	accountsFile, err := coininternal.LoadAccountsFile(accountsPath)
	if err != nil {
		return err
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("250"))
	fmt.Println(headerStyle.Render("account report"))
	fmt.Println()
	printAccountBalancesReport(accountsFile.Accounts)
	fmt.Println()
	printTotalBalancesReport(accountsFile.Accounts)
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
		rows = append(rows, table.Row{account.Name, account.Currency, coininternal.FormatMinor(account.BalanceMinor)})
	}

	renderTable(
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
		rows = append(rows, table.Row{currency, coininternal.FormatMinor(totals[currency])})
	}

	renderTable(
		[]table.Column{
			{Title: "CUR", Width: 5},
			{Title: "TOTAL", Width: 14},
		},
		rows,
	)
}

func printCategorySection(transactions []model.Transaction, start, end time.Time, showDetails bool) {
	fmt.Println(reportSectionStyle.Render("Range Categories"))
	fmt.Println()

	byCategory := make(map[string][]model.Transaction)
	for _, tx := range transactions {
		if tx.Type == coininternal.TransactionTypeTransfer {
			continue
		}
		inRange, err := coininternal.TransactionInRange(tx.Date, start, end)
		if err != nil || !inRange {
			continue
		}
		byCategory[tx.Category] = append(byCategory[tx.Category], tx)
	}

	if len(byCategory) == 0 {
		fmt.Println("  no transactions in range")
		return
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
			if tx.Type == coininternal.TransactionTypeExpense {
				currencyExpense[tx.Currency] += tx.AmountMinor
				report.expenseByCurrency[tx.Currency] += tx.AmountMinor
				report.totalExpenseMinor += tx.AmountMinor
			} else if tx.Type == coininternal.TransactionTypeIncome {
				currencyIncome[tx.Currency] += tx.AmountMinor
			}

			delta := tx.AmountMinor
			if tx.Type == coininternal.TransactionTypeExpense {
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
			totalRows = append(totalRows, table.Row{displayCategory, currency, coininternal.FormatMinor(report.totalsByCurrency[currency]), strconv.Itoa(len(report.items)), expenseShare})
		}
	}

	renderTable(
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
		summaryRows = append(summaryRows, table.Row{c, coininternal.FormatMinor(income), coininternal.FormatMinor(expense), coininternal.FormatMinor(net)})
	}

	if len(summaryRows) > 0 {
		fmt.Println()
		fmt.Println(reportSubSectionStyle.Render("Income / Expense Summary"))
		fmt.Println()
		renderTable(
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
	fmt.Println(reportSubSectionStyle.Render("Transactions By Category"))
	fmt.Println()
	if showDetails {
		renderSeparateCategoryDetails(categoryReports)
	} else {
		renderCompactCategoryDetails(categoryReports)
	}

	fmt.Println()
}

func renderCompactCategoryDetails(categoryReports []categoryReport) {

	detailRows := make([]table.Row, 0)
	for _, report := range categoryReports {
		items := make([]model.Transaction, len(report.items))
		copy(items, report.items)

		sort.Slice(items, func(i, j int) bool {
			if items[i].Date == items[j].Date {
				return items[i].CreatedAt > items[j].CreatedAt
			}
			return items[i].Date > items[j].Date
		})

		displayCategory := report.name
		if displayCategory == "" {
			displayCategory = "(no category)"
		}

		for idx, tx := range items {
			amount := coininternal.FormatMinor(tx.AmountMinor)
			if tx.Type == coininternal.TransactionTypeExpense {
				amount = "-" + amount
			}

			categoryCell := ""
			if idx == 0 {
				categoryCell = displayCategory
			}

			detailRows = append(detailRows, table.Row{categoryCell, tx.Date, amount, tx.Currency, tx.Account, tx.Note})
		}
	}

	renderTable(
		[]table.Column{
			{Title: "CATEGORY", Width: 18},
			{Title: "DATE", Width: 10},
			{Title: "AMOUNT", Width: 12},
			{Title: "CUR", Width: 5},
			{Title: "ACCOUNT", Width: 18},
			{Title: "NOTE", Width: 36},
		},
		detailRows,
	)
}

func renderSeparateCategoryDetails(categoryReports []categoryReport) {
	for i, report := range categoryReports {
		if i > 0 {
			fmt.Println()
		}

		items := make([]model.Transaction, len(report.items))
		copy(items, report.items)

		sort.Slice(items, func(i, j int) bool {
			if items[i].Date == items[j].Date {
				return items[i].CreatedAt > items[j].CreatedAt
			}
			return items[i].Date > items[j].Date
		})

		displayCategory := report.name
		if displayCategory == "" {
			displayCategory = "(no category)"
		}

		fmt.Println(categoryHeadingStyle(i).Render("- " + displayCategory))
		fmt.Println()

		rows := make([]table.Row, 0, len(items))
		for _, tx := range items {
			amount := coininternal.FormatMinor(tx.AmountMinor)
			if tx.Type == coininternal.TransactionTypeExpense {
				amount = "-" + amount
			}

			rows = append(rows, table.Row{tx.Date, amount, tx.Currency, tx.Account, tx.Note})
		}

		renderTable(
			[]table.Column{
				{Title: "DATE", Width: 10},
				{Title: "AMOUNT", Width: 12},
				{Title: "CUR", Width: 5},
				{Title: "ACCOUNT", Width: 18},
				{Title: "NOTE", Width: 36},
			},
			rows,
		)
	}
}

type categoryReport struct {
	name              string
	items             []model.Transaction
	totalsByCurrency  map[string]int64
	expenseByCurrency map[string]int64
	totalExpenseMinor int64
}

func categoryHeadingStyle(index int) lipgloss.Style {
	_ = index
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("180"))
}

func formatPercent(part, total int64) string {
	if total <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", (float64(part)*100)/float64(total))
}

func renderTable(columns []table.Column, rows []table.Row) {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(rows)+1),
	)

	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	styles.Cell = styles.Cell.Foreground(lipgloss.Color("252"))
	t.SetStyles(styles)

	fmt.Println(t.View())
}

func init() {
	reportCmd.Flags().BoolVar(&reportShowDetails, "details", false, "Show detailed transactions separated by category")
	rootCmd.AddCommand(reportCmd)
}
