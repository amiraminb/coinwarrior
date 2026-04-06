package cmd

import (
	"fmt"
	"sort"
	"time"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report <range>",
	Short: "Show balances and range category activity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		start, end, err := coininternal.ResolveDateRange(args[0], time.Now())
		if err != nil {
			return err
		}

		accountsPath, err := coininternal.FilePath(coininternal.AccountsFileName)
		if err != nil {
			return err
		}
		accountsFile, err := coininternal.LoadAccountsFile(accountsPath)
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

		fmt.Printf("report %s..%s\n\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
		printAccountBalances(accountsFile.Accounts)
		fmt.Println()
		printTotalBalances(accountsFile.Accounts)
		fmt.Println()
		printCategorySection(transactionsFile.Transactions, start, end)

		return nil
	},
}

func printAccountBalances(accounts []model.Account) {
	fmt.Println("Account Balances")
	if len(accounts) == 0 {
		fmt.Println("  no accounts")
		return
	}

	items := make([]model.Account, len(accounts))
	copy(items, accounts)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	for _, account := range items {
		fmt.Printf("- %s: %s %s\n", account.Name, account.Currency, coininternal.FormatMinor(account.BalanceMinor))
	}
}

func printTotalBalances(accounts []model.Account) {
	fmt.Println("Total Balances")
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

	for _, currency := range currencies {
		fmt.Printf("- %s %s\n", currency, coininternal.FormatMinor(totals[currency]))
	}
}

func printCategorySection(transactions []model.Transaction, start, end time.Time) {
	fmt.Println("Range Categories")

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

	for _, category := range categories {
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
		fmt.Printf("- %s\n", displayCategory)

		currencies := make([]string, 0, len(totals))
		for currency := range totals {
			currencies = append(currencies, currency)
		}
		sort.Strings(currencies)
		for _, currency := range currencies {
			fmt.Printf("  total %s %s\n", currency, coininternal.FormatMinor(totals[currency]))
		}

		sort.Slice(items, func(i, j int) bool {
			if items[i].Date == items[j].Date {
				return items[i].CreatedAt > items[j].CreatedAt
			}
			return items[i].Date > items[j].Date
		})

		for _, tx := range items {
			sign := "+"
			if tx.Type == coininternal.TransactionTypeExpense {
				sign = "-"
			}
			fmt.Printf("  %s | %s %s%s | %s | %s\n", tx.Date, tx.Currency, sign, coininternal.FormatMinor(tx.AmountMinor), tx.Account, tx.ID)
		}
	}
}

func init() {
	rootCmd.AddCommand(reportCmd)
}
