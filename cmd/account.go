package cmd

import (
	"fmt"
	"strings"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type accountAddStep int

const (
	accountStepName accountAddStep = iota
	accountStepCurrency
	accountStepOpeningBalance
	accountStepDone
)

type accountAddModel struct {
	step accountAddStep

	nameInput           string
	currencyInput       string
	openingBalanceInput string
}

var (
	accountFocusStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	accountMutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func newAccountAddModel() accountAddModel {
	return accountAddModel{
		step:                accountStepName,
		currencyInput:       "CAD",
		openingBalanceInput: "",
	}
}

func (m accountAddModel) Init() tea.Cmd {
	return nil
}

func (m accountAddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		switch m.step {
		case accountStepName:
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.nameInput) != "" {
					m.step = accountStepCurrency
				}
			case "backspace":
				if len(m.nameInput) > 0 {
					m.nameInput = m.nameInput[:len(m.nameInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.nameInput += msg.String()
				}
			}
		case accountStepCurrency:
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.currencyInput) != "" {
					m.step = accountStepOpeningBalance
				}
			case "esc":
				m.step = accountStepName
			case "backspace":
				if len(m.currencyInput) > 0 {
					m.currencyInput = m.currencyInput[:len(m.currencyInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					m.currencyInput += strings.ToUpper(msg.String())
				}
			}
		case accountStepOpeningBalance:
			switch msg.String() {
			case "enter":
				if strings.TrimSpace(m.openingBalanceInput) != "" {
					m.step = accountStepDone
					return m, tea.Quit
				}
			case "esc":
				m.step = accountStepCurrency
			case "backspace":
				if len(m.openingBalanceInput) > 0 {
					m.openingBalanceInput = m.openingBalanceInput[:len(m.openingBalanceInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					ch := msg.String()
					if (ch >= "0" && ch <= "9") || ch == "." {
						m.openingBalanceInput += ch
					}
				}
			}
		}
	}

	return m, nil
}

func (m accountAddModel) View() string {
	s := ""

	s += "Add Account\n\n"

	name := m.nameInput
	if m.step == accountStepName {
		name = accountFocusStyle.Render(name)
	}
	s += "Account name: " + name + "\n"

	currency := m.currencyInput
	if m.step == accountStepCurrency {
		currency = accountFocusStyle.Render(currency)
	}
	s += "Currency: " + currency + "\n"

	balance := m.openingBalanceInput
	if m.step == accountStepOpeningBalance {
		balance = accountFocusStyle.Render(balance)
	}
	s += "Opening balance: " + balance + "\n\n"

	s += accountMutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"

	return s
}

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage accounts",
}

var accountAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new account",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := runAccountAddInteractive()
		return err
	},
}

var accountUpdateCmd = &cobra.Command{
	Use:   "update <account> <amount>",
	Short: "Update account balance",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		account, err := coininternal.UpdateAccountBalance(args[0], args[1])
		if err != nil {
			return err
		}

		fmt.Printf("account updated: %s (%s %s)\n", account.Name, account.Currency, coininternal.FormatMinor(account.BalanceMinor))
		return nil
	},
}

func init() {
	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountUpdateCmd)
	rootCmd.AddCommand(accountCmd)
}

func runAccountAddInteractive() (bool, error) {
	model := newAccountAddModel()
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}

	result := finalModel.(accountAddModel)
	name := strings.TrimSpace(result.nameInput)
	if name == "" {
		fmt.Println("account add cancelled")
		return false, nil
	}

	currency := strings.TrimSpace(result.currencyInput)
	if currency == "" {
		currency = "CAD"
	}

	openingBalance := strings.TrimSpace(result.openingBalanceInput)
	if openingBalance == "" {
		openingBalance = "0"
	}

	account, err := coininternal.AddAccount(name, currency, openingBalance)
	if err != nil {
		return false, err
	}

	fmt.Printf("account created: %s (%s %s)\n", account.Name, account.Currency, coininternal.FormatMinor(account.BalanceMinor))
	return true, nil
}
