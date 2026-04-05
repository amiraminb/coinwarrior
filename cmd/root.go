package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "coinw",
	Short: "Coinwarrior CLI",
}

func Execute() error {
	return rootCmd.Execute()
}
