package cmd

import (
	"fmt"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a transaction",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		selected, ok, err := selectTransactionInteractive("Delete Transaction")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("delete cancelled")
			return nil
		}

		confirmed, err := runConfirmPrompt("Delete this transaction?\n" + formatEditableTransaction(selected))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("delete cancelled")
			return nil
		}

		tx, err := coininternal.DeleteTransaction(selected.ID)
		if err != nil {
			return err
		}

		fmt.Printf("deleted transaction: %s\n", tx.ID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
