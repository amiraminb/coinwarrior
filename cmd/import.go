package cmd

import (
	"github.com/amiraminb/coinwarrior/internal/importer"
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import transactions from a CSV file, categorizing each row interactively",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rows, err := importer.ParseFile(args[0])
		if err != nil {
			return err
		}
		return tui.RunImport(rows)
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
