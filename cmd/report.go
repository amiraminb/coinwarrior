package cmd

import (
	"fmt"
	"sort"
	"time"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
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
	fmt.Println("Range Categories")
	fmt.Println()

	byCategory := make(map[string][]model.Transaction)
	for _, tx := range transactions {
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

	for i, category := range categories {
		if i > 0 {
			fmt.Println()
		}

		items := byCategory[category]
		totals := make(map[string]int64)
		for _, tx := range items {
			delta := tx.AmountMinor
			if tx.Type == coininternal.TransactionTypeExpense {
				delta = -tx.AmountMinor
			}
			totals[tx.Currency] += delta
		}

		displayCategory := category
		if displayCategory == "" {
			displayCategory = "(no category)"
		}
		categoryStyle := categoryHeadingStyle(i)
		fmt.Printf("%s\n\n", categoryStyle.Render("- "+displayCategory))

		currencies := make([]string, 0, len(totals))
		for currency := range totals {
			currencies = append(currencies, currency)
		}
		sort.Strings(currencies)

		totalRows := make([]table.Row, 0, len(currencies))
		for _, currency := range currencies {
			totalRows = append(totalRows, table.Row{currency, coininternal.FormatMinor(totals[currency])})
		}
		renderTable(
			[]table.Column{
				{Title: "CUR", Width: 5},
				{Title: "CATEGORY TOTAL", Width: 18},
			},
			totalRows,
		)
		fmt.Println()

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
}

func categoryHeadingStyle(index int) lipgloss.Style {
	palette := []string{"81", "110", "150", "180", "117", "73"}
	color := palette[index%len(palette)]
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color))
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
