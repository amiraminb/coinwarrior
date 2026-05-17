package cmd

import (
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit a transaction",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunEditTransaction()
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
