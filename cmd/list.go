package cmd

import (
	"fmt"
	"sort"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/model"
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

		fmt.Println("ID | DATE | TYPE | AMOUNT | CURRENCY | CATEGORY | ACCOUNT")
		for _, tx := range items {
			fmt.Printf("%s | %s | %s | %s | %s | %s | %s\n",
				tx.ID,
				tx.Date,
				tx.Type,
				coininternal.FormatMinor(tx.AmountMinor),
				tx.Currency,
				tx.Category,
				tx.Account,
			)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
