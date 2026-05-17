package cmd

import (
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage accounts",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunAccountAction()
	},
}

func init() {
	rootCmd.AddCommand(accountCmd)
}
