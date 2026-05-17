package cmd

import (
	"fmt"
	"time"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/domain"
	"github.com/charmbracelet/bubbles/table"
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
		transactions, err := repo.LoadTransactions()
		if err != nil {
			return err
		}

		if len(transactions) == 0 {
			fmt.Println("no transactions")
			return nil
		}

		items := make([]domain.Transaction, len(transactions))
		copy(items, transactions)

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

		sortTransactionsByDateDesc(items)

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
			if tx.Type == domain.TransactionTypeTransfer {
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

		renderTable(columns, rows)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
