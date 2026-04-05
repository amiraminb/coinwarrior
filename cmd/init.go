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

		configPath := filepath.Join(dir, "config.json")
		configContent := []byte(`{
		  "schema_version": 1,
		  "default_currency": "CAD",
		  "timezone": "Local",
		  "date_format": "2006-01-02"
		}
		`)
		created, err := ensureFile(configPath, configContent)
		if err != nil {
			return err
		}
		if created {
			fmt.Println("created config.json")
		} else {
			fmt.Println("config.json exists")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
