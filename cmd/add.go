package cmd

import (
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/spf13/cobra"
)

var addCmd = newAddCommand()

func newAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a transaction",
		Long: `Add an expense or income transaction.

Supply any fields as flags and the TUI will prompt only for the missing fields.
Category and account names match existing names case-insensitively. To create
a new category or account, omit its flag and use the TUI.

Supply all fields to save without the entry form. Use --note "" for no note.
Possible duplicates still prompt for confirmation.`,
		Example: `  coinw add
  coinw add -t expense -a 20 -d 2026-05-21 -c Housing
  coinw add -t expense -a 20 -d 2026-05-21
  coinw add -t expense -a 20 -d 2026-05-21 -c Housing --account Checking --note ""`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			options := tui.AddTransactionOptions{}
			for _, field := range []struct {
				name  string
				value **string
			}{
				{"type", &options.Type},
				{"amount", &options.Amount},
				{"date", &options.Date},
				{"category", &options.Category},
				{"account", &options.Account},
				{"note", &options.Note},
			} {
				if cmd.Flags().Changed(field.name) {
					value, err := cmd.Flags().GetString(field.name)
					if err != nil {
						return err
					}
					*field.value = &value
				}
			}
			return tui.RunAddTransaction(options)
		},
	}
	cmd.Flags().StringP("type", "t", "", "Transaction type (expense or income)")
	cmd.Flags().StringP("amount", "a", "", "Amount (for example, 20 or 20.50)")
	cmd.Flags().StringP("date", "d", "", "Date (YYYY-MM-DD)")
	cmd.Flags().StringP("category", "c", "", "Existing category name")
	cmd.Flags().String("account", "", "Existing account name")
	cmd.Flags().StringP("note", "n", "", "Note (use an empty string for no note)")
	return cmd
}

func init() {
	rootCmd.AddCommand(addCmd)
}
