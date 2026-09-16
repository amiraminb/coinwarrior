package cmd

import (
	"fmt"

	"github.com/amiraminb/coinwarrior/internal/money"
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/spf13/cobra"
)

var futureCmd = &cobra.Command{
	Use:   "future <value>",
	Short: "Show a value's projected growth",
	Args:  cobra.ExactArgs(1),
	Example: `  coinw future 100
  coinw future 1,250.00`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFutureValue(args[0])
	},
}

func runFutureValue(input string) error {
	amountMinor, err := money.Parse(input)
	if err != nil {
		return err
	}
	if amountMinor <= 0 {
		return fmt.Errorf("value must be greater than zero")
	}

	rate := money.FutureValueInterestPercent
	fmt.Println(tui.HeaderStyle.Render(fmt.Sprintf("Future Value (%d%% Annual Interest)", rate)))
	fmt.Printf("Starting value: %s\n", money.Format(amountMinor))
	for _, years := range money.FutureValueHorizons() {
		futureMinor := money.FutureValueMinor(amountMinor, years)
		fmt.Printf("  %2d years: %s\n", years, money.Format(futureMinor))
	}

	return nil
}

func init() {
	rootCmd.AddCommand(futureCmd)
}
