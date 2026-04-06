package cmd

import (
	"fmt"
	"sort"
	"strconv"
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
)

var reportCmd = &cobra.Command{
	Use:   "report <range>",
	Short: "Show range category activity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
		printCategorySection(transactionsFile.Transactions, start, end)

		return nil
	},
}

func printCategorySection(transactions []model.Transaction, start, end time.Time) {
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

	categories := make([]string, 0, len(byCategory))
	for category := range byCategory {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	fmt.Println(reportSubSectionStyle.Render("Category Totals (Range)"))
	fmt.Println()
	totalRows := make([]table.Row, 0)
	currencyIncome := make(map[string]int64)
	currencyExpense := make(map[string]int64)
	for _, category := range categories {
		items := byCategory[category]
		for _, tx := range items {
			if tx.Type == coininternal.TransactionTypeExpense {
				currencyExpense[tx.Currency] += tx.AmountMinor
			} else if tx.Type == coininternal.TransactionTypeIncome {
				currencyIncome[tx.Currency] += tx.AmountMinor
			}
		}

		totals := make(map[string]int64)
		for _, tx := range items {
			delta := tx.AmountMinor
			if tx.Type == coininternal.TransactionTypeExpense {
				delta = -tx.AmountMinor
			}
			totals[tx.Currency] += delta
		}

		currencies := make([]string, 0, len(totals))
		for currency := range totals {
			currencies = append(currencies, currency)
		}
		sort.Strings(currencies)

		displayCategory := category
		if displayCategory == "" {
			displayCategory = "(no category)"
		}

		for _, currency := range currencies {
			totalRows = append(totalRows, table.Row{displayCategory, currency, coininternal.FormatMinor(totals[currency]), strconv.Itoa(len(items))})
		}
	}

	renderTable(
		[]table.Column{
			{Title: "CATEGORY", Width: 20},
			{Title: "CUR", Width: 5},
			{Title: "TOTAL", Width: 14},
			{Title: "TXNS", Width: 6},
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

	for i, category := range categories {
		if i > 0 {
			fmt.Println()
		}

		items := byCategory[category]

		displayCategory := category
		if displayCategory == "" {
			displayCategory = "(no category)"
		}
		categoryStyle := categoryHeadingStyle(i)
		fmt.Printf("%s\n\n", categoryStyle.Render("- "+displayCategory))

		sort.Slice(items, func(i, j int) bool {
			if items[i].Date == items[j].Date {
				return items[i].CreatedAt > items[j].CreatedAt
			}
			return items[i].Date > items[j].Date
		})

		rows := make([]table.Row, 0, len(items))
		for _, tx := range items {
			amount := coininternal.FormatMinor(tx.AmountMinor)
			if tx.Type == coininternal.TransactionTypeExpense {
				amount = "-" + amount
			}
			rows = append(rows, table.Row{tx.Date, tx.Type, amount, tx.Currency, tx.Account, tx.Note, tx.ID})
		}

		renderTable(
			[]table.Column{
				{Title: "DATE", Width: 10},
				{Title: "TYPE", Width: 8},
				{Title: "AMOUNT", Width: 12},
				{Title: "CUR", Width: 5},
				{Title: "ACCOUNT", Width: 18},
				{Title: "NOTE", Width: 20},
				{Title: "ID", Width: 24},
			},
			rows,
		)
	}

	fmt.Println()
}

func categoryHeadingStyle(index int) lipgloss.Style {
	_ = index
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("180"))
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
	rootCmd.AddCommand(reportCmd)
}
