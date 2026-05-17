package cmd

import (
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/spf13/cobra"
)

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Manage monthly budgets",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunBudgetAction()
	},
}

func init() {
	rootCmd.AddCommand(budgetCmd)
}
