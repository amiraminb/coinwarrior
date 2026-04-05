package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		dir := filepath.Join(home, "Documents", ".coinwarrior")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}

		fmt.Printf("data dir ready: %s\n", dir)

		err = createConfigFile(filepath.Join(dir, "config.json"), []byte(`{
		  "schema_version": 1,
		  "default_currency": "CAD",
		  "timezone": "Local",
		  "date_format": "2006-01-02"
		}
		`))
		if err != nil {
			return err
		}

		err = createConfigFile(filepath.Join(dir, "transactions.json"), []byte(`{
		  "schema_version": 1,
		  "transactions": []
		}
		`))
		if err != nil {
			return err
		}

		err = createConfigFile(filepath.Join(dir, "budgets.json"), []byte(`{
		  "schema_version": 1,
		  "budgets": []
		}
		`))
		if err != nil {
			return err
		}

		err = createConfigFile(filepath.Join(dir, "recurring.json"), []byte(`{
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
