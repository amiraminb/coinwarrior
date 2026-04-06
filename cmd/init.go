package cmd

import (
	"fmt"
	"os"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	tea "github.com/charmbracelet/bubbletea"
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

		if err := setupInitialAccounts(accountsPath); err != nil {
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

func setupInitialAccounts(accountsPath string) error {
	accountsFile, err := coininternal.LoadAccountsFile(accountsPath)
	if err != nil {
		return err
	}
	if len(accountsFile.Accounts) > 0 {
		return nil
	}

	addNow, err := runConfirmPrompt("No accounts found. Add one now?")
	if err != nil {
		return err
	}
	if !addNow {
		return nil
	}

	for {
		_, err := runAccountAddInteractive()
		if err != nil {
			return err
		}

		again, err := runConfirmPrompt("Add another account?")
		if err != nil {
			return err
		}
		if !again {
			break
		}
	}

	return nil
}

type confirmModel struct {
	question string
	cursor   int
	answer   bool
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "left", "h", "up", "k":
			m.cursor = 0
		case "right", "l", "down", "j":
			m.cursor = 1
		case "enter":
			m.answer = m.cursor == 0
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m confirmModel) View() string {
	s := m.question + "\n\n"

	yes := "  Yes"
	no := "  No"
	if m.cursor == 0 {
		yes = accountFocusStyle.Render("> Yes")
	} else {
		no = accountFocusStyle.Render("> No")
	}

	s += yes + "\n"
	s += no + "\n\n"
	s += accountMutedStyle.Render("(use ←/→ or ↑/↓ and enter)") + "\n"

	return s
}

func runConfirmPrompt(question string) (bool, error) {
	p := tea.NewProgram(confirmModel{question: question})

	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}

	result := finalModel.(confirmModel)
	return result.answer, nil
}
