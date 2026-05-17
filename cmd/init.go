package cmd

import (
	"fmt"
	"os"

	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/spf13/cobra"
)

func createConfigFile(repo *repository.FileRepository, name string, content []byte) error {
	path, created, err := repo.CreateFile(name, content)
	if err != nil {
		return err
	}
	if created {
		fmt.Printf("created %s\n", path)
	} else {
		fmt.Printf("%s exists\n", path)
	}

	return nil
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize coinwarrior data",
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := repository.NewFileRepository()
		dir, err := repo.DataDir()
		if err != nil {
			return err
		}

		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}

		fmt.Printf("data dir ready: %s\n", dir)

		err = createConfigFile(repo, repository.ConfigFileName, []byte(`{
		  "schema_version": 1,
		  "default_currency": "CAD",
		  "timezone": "Local",
		  "date_format": "2006-01-02"
		}
		`))
		if err != nil {
			return err
		}

		err = createConfigFile(repo, repository.TransactionsFileName, []byte(`{
		  "schema_version": 1,
		  "transactions": []
		}
		`))
		if err != nil {
			return err
		}

		err = createConfigFile(repo, repository.AccountsFileName, []byte(`{
		  "schema_version": 1,
		  "accounts": []
		}
		`))
		if err != nil {
			return err
		}

		if err := setupInitialAccounts(repo); err != nil {
			return err
		}

		err = createConfigFile(repo, repository.CategoriesFileName, []byte(`{
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

		err = createConfigFile(repo, repository.BudgetsFileName, []byte(`{
		  "schema_version": 1,
		  "budgets": []
		}
		`))
		if err != nil {
			return err
		}

		err = createConfigFile(repo, repository.RecurringFileName, []byte(`{
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

func setupInitialAccounts(repo repository.Repository) error {
	accounts, err := repo.LoadAccounts()
	if err != nil {
		return err
	}
	if len(accounts) > 0 {
		return nil
	}

	addNow, err := tui.RunConfirmPrompt("No accounts found. Add one now?")
	if err != nil {
		return err
	}
	if !addNow {
		return nil
	}

	for {
		_, err := tui.RunAccountAdd()
		if err != nil {
			return err
		}

		again, err := tui.RunConfirmPrompt("Add another account?")
		if err != nil {
			return err
		}
		if !again {
			break
		}
	}

	return nil
}

