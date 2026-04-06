package cmd

import (
	"fmt"
	"sort"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List transactions",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := coininternal.FilePath(coininternal.TransactionsFileName)
		if err != nil {
			return err
		}

		transactions, err := coininternal.LoadTransactions(path)
		if err != nil {
			return err
		}

		if len(transactions.Transactions) == 0 {
			fmt.Println("no transactions")
			return nil
		}

		items := make([]model.Transaction, len(transactions.Transactions))
		copy(items, transactions.Transactions)

		sort.Slice(items, func(i, j int) bool {
			if items[i].Date == items[j].Date {
				return items[i].CreatedAt > items[j].CreatedAt
			}
			return items[i].Date > items[j].Date
		})

		columns := []table.Column{
			{Title: "ID", Width: 24},
			{Title: "DATE", Width: 10},
			{Title: "TYPE", Width: 8},
			{Title: "AMOUNT", Width: 12},
			{Title: "CUR", Width: 5},
			{Title: "CATEGORY", Width: 16},
			{Title: "ACCOUNT", Width: 16},
		}

		rows := make([]table.Row, 0, len(items))
		for _, tx := range items {
			rows = append(rows, table.Row{
				tx.ID,
				tx.Date,
				tx.Type,
				coininternal.FormatMinor(tx.AmountMinor),
				tx.Currency,
				tx.Category,
				tx.Account,
			})
		}

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
		styles.Cell = styles.Cell.
			Foreground(lipgloss.Color("252"))
		t.SetStyles(styles)

		fmt.Println(t.View())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
