package cmd

import (
	"fmt"
	"sort"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
)

var accountReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show current account balances",
	RunE: func(cmd *cobra.Command, args []string) error {
		accountsPath, err := coininternal.FilePath(coininternal.AccountsFileName)
		if err != nil {
			return err
		}

		accountsFile, err := coininternal.LoadAccountsFile(accountsPath)
		if err != nil {
			return err
		}

		fmt.Println("account report")
		fmt.Println()
		printAccountBalances(accountsFile.Accounts)
		fmt.Println()
		printTotalBalances(accountsFile.Accounts)

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

func init() {
	accountCmd.AddCommand(accountReportCmd)
}
