package cmd

import (
	"fmt"
	"sort"
	"time"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/domain"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [range]",
	Short: "List transactions",
	Long: `List transactions.

Supported ranges: today, yesterday, week, lastweek, month, lastmonth, year, lastyear, or YYYY-MM-DD..YYYY-MM-DD.`,
	Example: `  coinw list
  coinw list month
  coinw list yesterday
  coinw list 2026-04-01..2026-04-30`,
	Args: cobra.MaximumNArgs(1),
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

		items := make([]domain.Transaction, len(transactions.Transactions))
		copy(items, transactions.Transactions)

		if len(args) == 1 {
			start, end, err := coininternal.ResolveDateRange(args[0], time.Now())
			if err != nil {
				return err
			}

			filtered := make([]domain.Transaction, 0, len(items))
			for _, tx := range items {
				inRange, err := coininternal.TransactionInRange(tx.Date, start, end)
				if err != nil {
					return fmt.Errorf("invalid transaction date '%s' for %s", tx.Date, tx.ID)
				}
				if inRange {
					filtered = append(filtered, tx)
				}
			}

			items = filtered
		}

		if len(items) == 0 {
			fmt.Println("no transactions")
			return nil
		}

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
			{Title: "NOTE", Width: 20},
		}

		rows := make([]table.Row, 0, len(items))
		for _, tx := range items {
			account := tx.Account
			if tx.Type == coininternal.TransactionTypeTransfer {
				account = tx.Account + " -> " + tx.ToAccount
			}
			rows = append(rows, table.Row{
				tx.ID,
				tx.Date,
				tx.Type,
				coininternal.FormatMinor(tx.AmountMinor),
				tx.Currency,
				tx.Category,
				account,
				tx.Note,
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
