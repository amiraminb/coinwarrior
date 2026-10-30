package cmd

import (
	"fmt"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/spf13/cobra"
)

var budgetCmd = newBudgetCommand()

func newBudgetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budget",
		Short: "Manage monthly overall and category budgets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.RunBudgetAction()
		},
	}
	show := &cobra.Command{
		Use:   "show [month]",
		Short: "Show overall and category budgets (defaults to this month)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			month := ""
			if len(args) == 1 {
				month = args[0]
			}
			return tui.RunBudgetShow(month)
		},
	}
	set := &cobra.Command{
		Use:   "set [category] <month> <amount>",
		Short: "Set a monthly category limit, or an overall limit without a category",
		Example: `  coinw budget set Groceries 2026-11 500
  coinw budget set "Personal Care" 2026-11 150
  coinw budget set 2026-11 4500
  coinw budget set Travel 2026-12 400 --carryover`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			category := ""
			if len(args) == 3 {
				category = args[0]
				args = args[1:]
			}
			currency, err := cmd.Flags().GetString("currency")
			if err != nil {
				return err
			}
			var budget model.Budget
			if cmd.Flags().Changed("carryover") {
				carry, err := cmd.Flags().GetBool("carryover")
				if err != nil {
					return err
				}
				budget, err = svc.SetMonthlyCategoryBudgetWithCarryover(args[0], currency, category, args[1], carry)
				if err != nil {
					return err
				}
			} else {
				budget, err = svc.SetMonthlyCategoryBudget(args[0], currency, category, args[1])
				if err != nil {
					return err
				}
			}
			fmt.Printf("budget set: %s / %s (%s %s)\n", budget.Month, tui.BudgetCategoryLabel(budget.Category), budget.Currency, money.Format(budget.AmountMinor))
			return nil
		},
	}
	set.Flags().String("currency", "CAD", "Budget currency")
	set.Flags().Bool("carryover", false, "Carry the previous month's remaining budget for the same category; use --carryover=false to skip")
	cmd.AddCommand(show, set)
	return cmd
}

func init() {
	rootCmd.AddCommand(budgetCmd)
}
