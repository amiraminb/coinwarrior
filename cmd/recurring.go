package cmd

import (
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/spf13/cobra"
)

var recurringCmd = &cobra.Command{
	Use:   "recurring",
	Short: "Manage recurring transactions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunRecurringAction()
	},
}

func init() {
	rootCmd.AddCommand(recurringCmd)
}
