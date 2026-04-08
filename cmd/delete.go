package cmd

import (
	"fmt"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type deleteModel struct {
	transactions []model.Transaction
	cursor       int
	selected     model.Transaction
}

func newDeleteModel(transactions []model.Transaction) deleteModel {
	items := make([]model.Transaction, len(transactions))
	copy(items, transactions)

	return deleteModel{transactions: items}
}

func (m deleteModel) Init() tea.Cmd {
	return nil
}

func (m deleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.transactions)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.transactions) == 0 {
				break
			}
			m.selected = m.transactions[m.cursor]
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m deleteModel) View() string {
	s := "Delete Transaction\n\n"
	s += "Select transaction:\n\n"

	for i, tx := range m.transactions {
		line := "  " + formatEditableTransaction(tx)
		if i == m.cursor {
			line = editFocusStyle.Render("> " + formatEditableTransaction(tx))
		}
		s += line + "\n"
	}

	s += "\n" + editMutedStyle.Render("(use ↑/↓ and enter, esc to cancel, q to quit)") + "\n"
	return s
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a transaction",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		transactionsPath, err := coininternal.FilePath(coininternal.TransactionsFileName)
		if err != nil {
			return err
		}

		transactionsFile, err := coininternal.LoadTransactions(transactionsPath)
		if err != nil {
			return err
		}
		if len(transactionsFile.Transactions) == 0 {
			return fmt.Errorf("no transactions available")
		}

		sortEditableTransactions(transactionsFile.Transactions)

		p := tea.NewProgram(newDeleteModel(transactionsFile.Transactions))
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		result := finalModel.(deleteModel)
		if result.selected.ID == "" {
			fmt.Println("delete cancelled")
			return nil
		}

		confirmed, err := runConfirmPrompt("Delete this transaction?\n" + formatEditableTransaction(result.selected))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("delete cancelled")
			return nil
		}

		tx, err := coininternal.DeleteTransaction(result.selected.ID)
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
