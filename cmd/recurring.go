package cmd

import (
	"fmt"

	"github.com/amiraminb/coinwarrior/internal/service"
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/spf13/cobra"
)

var recurringCmd = newRecurringCommand()

func newRecurringCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "recurring", Short: "Manage recurring transactions", Args: cobra.NoArgs,
		Long: `Manage monthly recurring rules. With no subcommand, open the TUI menu.
Use list (or -l/--list) to print rules with notes, monthly totals, and future values.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			list, err := cmd.Flags().GetBool("list")
			if err != nil {
				return err
			}
			if list {
				return tui.RunRecurringList()
			}
			return tui.RunRecurringAction()
		},
	}
	cmd.Flags().BoolP("list", "l", false, "List recurring rules with notes")
	cmd.AddCommand(&cobra.Command{
		Use: "list", Short: "List recurring rules with notes", Args: cobra.NoArgs,
		Long: `List all saved recurring rules with notes, followed by monthly income,
expense, and net totals per currency. Transfers are excluded from totals.
Future values project one month's listed totals as one-time amounts at 5%
annual compound interest over 5, 10, 15, and 20 years.`,
		RunE: func(cmd *cobra.Command, args []string) error { return tui.RunRecurringList() },
	})
	add := &cobra.Command{
		Use: "add", Short: "Add a recurring rule using flags or the TUI", Args: cobra.NoArgs,
		Long: `Add a monthly recurring rule. Without flags, open the full TUI form.
With flags, prompt only for missing type, amount, category, account, and day.
Transfers use --to-account instead of --category. Currency defaults to CAD,
start date to today, and end date and note to empty. Supply all required fields
to save directly. Category and account names match saved names case-insensitively.`,
		Example: `  coinw recurring add
  coinw recurring add -t expense -a 14.16 -c Fitness --account Checking --day 5 --note "Carbon Diet"
  coinw recurring add -t transfer -a 100 --account Checking --to-account Savings --day 15`,
		RunE: func(cmd *cobra.Command, args []string) error {
			options, err := recurringOptionsFromFlags(cmd)
			if err != nil {
				return err
			}
			return tui.RunRecurringAdd(options)
		},
	}
	edit := &cobra.Command{
		Use: "edit [id]", Short: "Edit a recurring rule using flags or the TUI", Args: cobra.MaximumNArgs(1),
		Long: `Edit a recurring rule. Omit the ID to pick a rule interactively.
With flags, update only the supplied fields and keep the rest. Without flags,
open the edit form. Use --note "" or --end-date "" to clear those fields.`,
		Example: `  coinw recurring edit
  coinw recurring edit rec_123 --amount 19.99 --note "New price"`,
		ValidArgsFunction: completeRecurringRuleID,
		RunE: func(cmd *cobra.Command, args []string) error {
			options, err := recurringOptionsFromFlags(cmd)
			if err != nil {
				return err
			}
			return tui.RunRecurringEdit(recurringIDArgument(args), options)
		},
	}
	for _, command := range []*cobra.Command{add, edit} {
		command.Flags().StringP("type", "t", "", "Transaction type (expense, income, or transfer)")
		command.Flags().StringP("amount", "a", "", "Amount (for example, 20.50)")
		command.Flags().String("currency", "", "Currency (defaults to CAD for add)")
		command.Flags().StringP("category", "c", "", "Existing category name")
		command.Flags().String("account", "", "Existing account name (source for transfers)")
		command.Flags().String("to-account", "", "Destination account for transfers")
		command.Flags().IntP("day", "d", 0, "Day of month (1-28)")
		command.Flags().String("start-date", "", "Start date, YYYY-MM-DD (defaults to today for add)")
		command.Flags().String("end-date", "", "End date, YYYY-MM-DD (empty for no end date)")
		command.Flags().StringP("note", "n", "", "Note (empty for no note)")
	}
	cmd.AddCommand(add, edit, &cobra.Command{
		Use: "delete [id]", Short: "Delete a recurring rule by ID, or choose one in the TUI", Args: cobra.MaximumNArgs(1),
		Long: `Delete the rule with the given ID directly. Omit the ID to select and
confirm deletion in the TUI. Previously generated transactions are kept.`,
		ValidArgsFunction: completeRecurringRuleID,
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.RunRecurringDelete(recurringIDArgument(args))
		},
	}, &cobra.Command{
		Use: "generate", Short: "Generate due recurring transactions through the current month", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error { return tui.RunRecurringGenerate() },
	})
	return cmd
}

func recurringOptionsFromFlags(cmd *cobra.Command) (service.RecurringRuleEdits, error) {
	options := service.RecurringRuleEdits{}
	for _, field := range []struct {
		name  string
		value **string
	}{
		{"type", &options.Type}, {"amount", &options.AmountInput},
		{"currency", &options.Currency}, {"category", &options.Category},
		{"account", &options.Account}, {"to-account", &options.ToAccount},
		{"start-date", &options.StartDate}, {"end-date", &options.EndDate}, {"note", &options.Note},
	} {
		if cmd.Flags().Changed(field.name) {
			value, err := cmd.Flags().GetString(field.name)
			if err != nil {
				return options, err
			}
			*field.value = &value
		}
	}
	if cmd.Flags().Changed("day") {
		day, err := cmd.Flags().GetInt("day")
		if err != nil {
			return options, err
		}
		options.DayOfMonth = &day
	}
	return options, nil
}

func recurringIDArgument(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func completeRecurringRuleID(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	rules, err := svc.LoadRecurringRules()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	items := make([]string, 0, len(rules))
	for _, rule := range rules {
		items = append(items, fmt.Sprintf("%s\t%s %s", rule.ID, rule.Category, rule.Note))
	}
	return items, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(recurringCmd)
}
