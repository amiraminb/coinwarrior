package cmd

import (
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a transaction",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunAddTransaction()
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
