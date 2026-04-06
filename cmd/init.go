package cmd

import (
	"fmt"
	"os"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/spf13/cobra"
)

func ensureFile(path string, content []byte) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func createConfigFile(configPath string, content []byte) error {
	created, err := ensureFile(configPath, content)
	if err != nil {
		return err
	}
	if created {
		fmt.Printf("created %s\n", configPath)
	} else {
		fmt.Printf("%s exists\n", configPath)
	}

	return nil
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize coinwarrior data",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := coininternal.DataDir()
		if err != nil {
			return err
		}

		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}

		fmt.Printf("data dir ready: %s\n", dir)

		configPath, err := coininternal.FilePath(coininternal.ConfigFileName)
		if err != nil {
			return err
		}

		err = createConfigFile(configPath, []byte(`{
		  "schema_version": 1,
		  "default_currency": "CAD",
		  "timezone": "Local",
		  "date_format": "2006-01-02"
		}
		`))
		if err != nil {
			return err
		}

		transactionsPath, err := coininternal.FilePath(coininternal.TransactionsFileName)
		if err != nil {
			return err
		}

		err = createConfigFile(transactionsPath, []byte(`{
		  "schema_version": 1,
		  "transactions": []
		}
		`))
		if err != nil {
			return err
		}

		accountsPath, err := coininternal.FilePath(coininternal.AccountsFileName)
		if err != nil {
			return err
		}

		err = createConfigFile(accountsPath, []byte(`{
		  "schema_version": 1,
		  "accounts": []
		}
		`))
		if err != nil {
			return err
		}

		categoriesPath, err := coininternal.FilePath(coininternal.CategoriesFileName)
		if err != nil {
			return err
		}

		err = createConfigFile(categoriesPath, []byte(`{
		  "schema_version": 1,
		  "categories": [
		    "Housing",
		    "Utilities",
		    "Groceries",
		    "Dining",
		    "Transportation",
		    "Healthcare",
		    "Insurance",
		    "Subscriptions",
		    "Entertainment",
		    "Income"
		  ]
		}
		`))
		if err != nil {
			return err
		}

		budgetsPath, err := coininternal.FilePath(coininternal.BudgetsFileName)
		if err != nil {
			return err
		}

		err = createConfigFile(budgetsPath, []byte(`{
		  "schema_version": 1,
		  "budgets": []
		}
		`))
		if err != nil {
			return err
		}

		recurringPath, err := coininternal.FilePath(coininternal.RecurringFileName)
		if err != nil {
			return err
		}

		err = createConfigFile(recurringPath, []byte(`{
		  "schema_version": 1,
		  "recurring_rules": []
		}
		`))
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
