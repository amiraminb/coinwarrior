package cmd

import (
	"fmt"
	"strings"

	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/domain"
	"github.com/amiraminb/coinwarrior/internal/repository"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type (
	accountAddStep    int
	accountUpdateStep int
	accountAction     string
)

const (
	accountActionAdd    accountAction = "add"
	accountActionUpdate accountAction = "update"
	accountActionQuit   accountAction = "quit"
)

const (
	accountStepName accountAddStep = iota
	accountStepCurrency
	accountStepOpeningBalance
	accountStepDone
)

const (
	accountUpdateStepSelect accountUpdateStep = iota
	accountUpdateStepAmount
	accountUpdateStepConfirm
	accountUpdateStepDone
)

type accountAddModel struct {
	step accountAddStep

	nameInput           string
	currencyInput       string
	openingBalanceInput string
}

type accountUpdateModel struct {
	step accountUpdateStep

	accounts []domain.Account
	cursor   int

	selectedAccount domain.Account
	amountInput     string
	confirmCursor   int
	confirmed       bool
	errMessage      string
}

func newAccountAddModel() accountAddModel {
	return accountAddModel{
		step:                accountStepName,
		currencyInput:       "CAD",
		openingBalanceInput: "",
	}
}

func newAccountUpdateModel(accounts []domain.Account) accountUpdateModel {
	return accountUpdateModel{
		step:     accountUpdateStepSelect,
		accounts: accounts,
	}
}

func (m accountAddModel) Init() tea.Cmd {
	return nil
}

func (m accountUpdateModel) Init() tea.Cmd {
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

func (m accountUpdateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		switch m.step {
		case accountUpdateStepSelect:
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.accounts)-1 {
					m.cursor++
				}
			case "enter":
				if len(m.accounts) == 0 {
					break
				}
				m.selectedAccount = m.accounts[m.cursor]
				m.amountInput = ""
				m.confirmCursor = 0
				m.errMessage = ""
				m.step = accountUpdateStepAmount
			}
		case accountUpdateStepAmount:
			switch msg.String() {
			case "enter":
				if _, err := coininternal.ParseAmount(m.amountInput); err != nil {
					m.errMessage = err.Error()
					break
				}
				m.errMessage = ""
				m.confirmCursor = 0
				m.step = accountUpdateStepConfirm
			case "esc":
				m.errMessage = ""
				m.step = accountUpdateStepSelect
			case "backspace":
				m.errMessage = ""
				if len(m.amountInput) > 0 {
					m.amountInput = m.amountInput[:len(m.amountInput)-1]
				}
			default:
				if len(msg.String()) == 1 {
					ch := msg.String()
					if (ch >= "0" && ch <= "9") || ch == "." || ch == "-" {
						m.amountInput += ch
						m.errMessage = ""
					}
				}
			}
		case accountUpdateStepConfirm:
			switch msg.String() {
			case "left", "h", "up", "k":
				m.confirmCursor = 0
			case "right", "l", "down", "j":
				m.confirmCursor = 1
			case "enter":
				if m.confirmCursor == 0 {
					m.confirmed = true
					m.step = accountUpdateStepDone
					return m, tea.Quit
				}
				m.step = accountUpdateStepAmount
			case "esc":
				m.step = accountUpdateStepAmount
			}
		}
	}

	return m, nil
}

func (m accountAddModel) View() string {
	s := ""

	s += "Add Account\n\n"

	if m.step == accountStepName {
		s += renderActiveField("Account name: ", m.nameInput) + "\n"
	} else {
		s += renderField("Account name: ", m.nameInput) + "\n"
	}

	if m.step == accountStepCurrency {
		s += renderActiveField("Currency: ", m.currencyInput) + "\n"
	} else {
		s += renderField("Currency: ", m.currencyInput) + "\n"
	}

	if m.step == accountStepOpeningBalance {
		s += renderActiveField("Opening balance: ", m.openingBalanceInput) + "\n\n"
	} else {
		s += renderField("Opening balance: ", m.openingBalanceInput) + "\n\n"
	}

	s += mutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"

	return s
}

func (m accountUpdateModel) View() string {
	s := "Update Account Balance\n\n"

	switch m.step {
	case accountUpdateStepSelect:
		s += "Select account:\n\n"
		for i, account := range m.accounts {
			line := fmt.Sprintf("  %s (%s %s)", account.Name, account.Currency, coininternal.FormatMinor(account.BalanceMinor))
			if i == m.cursor {
				line = focusStyle.Render(fmt.Sprintf("> %s (%s %s)", account.Name, account.Currency, coininternal.FormatMinor(account.BalanceMinor)))
			}
			s += line + "\n"
		}
		s += "\n" + mutedStyle.Render("(use ↑/↓ and enter, q to quit)") + "\n"
	case accountUpdateStepAmount:
		s += renderField("Account: ", m.selectedAccount.Name) + "\n"
		s += renderField("Currency: ", m.selectedAccount.Currency) + "\n"
		s += renderField("Current balance: ", coininternal.FormatMinor(m.selectedAccount.BalanceMinor)) + "\n\n"
		s += renderActiveField("Enter new balance: ", m.amountInput) + "\n"
		if m.errMessage != "" {
			s += warnStyle.Render(m.errMessage) + "\n"
		}
		s += "\n" + mutedStyle.Render("(enter to continue, esc to go back, q to quit)") + "\n"
	case accountUpdateStepConfirm:
		newBalanceMinor, _ := coininternal.ParseAmount(m.amountInput)
		s += renderField("Account: ", m.selectedAccount.Name) + "\n"
		s += renderField("Currency: ", m.selectedAccount.Currency) + "\n"
		s += renderField("Current balance: ", coininternal.FormatMinor(m.selectedAccount.BalanceMinor)) + "\n"
		s += renderField("New balance: ", coininternal.FormatMinor(newBalanceMinor)) + "\n\n"
		s += warnStyle.Render("Confirm account balance update?") + "\n\n"

		yes := "  Yes"
		no := "  No"
		if m.confirmCursor == 0 {
			yes = focusStyle.Render("> Yes")
		} else {
			no = focusStyle.Render("> No")
		}

		s += yes + "\n"
		s += no + "\n\n"
		s += mutedStyle.Render("(use ←/→ or ↑/↓, enter to confirm, esc to go back, q to quit)") + "\n"
	case accountUpdateStepDone:
		s += mutedStyle.Render("Done") + "\n"
	}

	return s
}

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage accounts",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		action, err := runAccountMenuInteractive()
		if err != nil {
			return err
		}

		switch action {
		case accountActionAdd:
			_, err := runAccountAddInteractive()
			return err
		case accountActionUpdate:
			_, err := runAccountUpdateInteractive()
			return err
		case accountActionQuit:
			return nil
		default:
			fmt.Println("account cancelled")
			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(accountCmd)
}

func runAccountMenuInteractive() (accountAction, error) {
	items := []selectionItem[accountAction]{
		{label: "Add account", value: accountActionAdd},
		{label: "Update account balance", value: accountActionUpdate},
		{label: "Quit", value: accountActionQuit},
	}
	action, _, err := runSelection("Account", "Choose an action:", items)
	if err != nil {
		return "", err
	}
	return action, nil
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

func runAccountUpdateInteractive() (bool, error) {
	accounts, err := repository.FRepository.LoadAccounts()
	if err != nil {
		return false, err
	}
	if len(accounts) == 0 {
		return false, fmt.Errorf("no accounts available; create one with 'coinw account'")
	}

	p := tea.NewProgram(newAccountUpdateModel(accounts))
	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}

	result := finalModel.(accountUpdateModel)
	if !result.confirmed || strings.TrimSpace(result.selectedAccount.Name) == "" {
		fmt.Println("account update cancelled")
		return false, nil
	}

	account, err := coininternal.UpdateAccountBalance(result.selectedAccount.Name, result.amountInput)
	if err != nil {
		return false, err
	}

	fmt.Printf("account updated: %s (%s %s)\n", account.Name, account.Currency, coininternal.FormatMinor(account.BalanceMinor))
	return true, nil
}
