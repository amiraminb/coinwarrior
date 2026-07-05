package cmd

import (
	"fmt"
	"time"

	"github.com/amiraminb/coinwarrior/internal/daterange"
	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
)

var reportTransactionsCmd = &cobra.Command{
	Use:   "transactions <range>",
	Short: "List transactions in a range",
	Long: `List transactions in a range.

Supported ranges: today, yesterday, week, lastweek, month, lastmonth, year, lastyear, or YYYY-MM-DD..YYYY-MM-DD.`,
	Example: `  coinw report transactions month
  coinw report transactions yesterday
  coinw report transactions 2026-04-01..2026-04-30`,
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

		items := make([]model.Transaction, 0, len(transactions))
		for _, tx := range transactions {
			inRange, err := daterange.Contains(tx.Date, start, end)
			if err != nil {
				return fmt.Errorf("invalid transaction date '%s' for %s", tx.Date, tx.ID)
			}
			if inRange {
				items = append(items, tx)
			}
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

		return nil
	},
}
