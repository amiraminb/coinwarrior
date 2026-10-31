package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/spf13/cobra"
)

func TestExecuteWarnsAboutPendingRecurringOnEveryInvocation(t *testing.T) {
	r := setupRecurringRepository(t)
	now := time.Now()
	rule := model.RecurringRule{
		ID: "rec_test", Type: "expense", AmountMinor: 1000, Currency: "CAD", Category: "Utilities",
		Account: "Checking", DayOfMonth: 28, StartDate: now.Format("2006-01") + "-01",
	}
	if err := r.SaveRecurringRules([]model.RecurringRule{rule}); err != nil {
		t.Fatal(err)
	}
	previousRoot := rootCmd
	t.Cleanup(func() { rootCmd = previousRoot })
	for _, args := range [][]string{{"noop"}, {"--help"}, {"--version"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			rootCmd = &cobra.Command{Use: "coinw", Version: "test"}
			called := false
			rootCmd.AddCommand(&cobra.Command{Use: "noop", Run: func(cmd *cobra.Command, args []string) { called = true }})
			var warnings bytes.Buffer
			rootCmd.SetErr(&warnings)
			rootCmd.SetOut(io.Discard)
			rootCmd.SetArgs(args)
			if err := Execute(); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(warnings.String(), "1 recurring transaction(s)") || !strings.Contains(warnings.String(), "coinw recurring generate") {
				t.Fatalf("missing reminder: %s", warnings.String())
			}
			if args[0] == "noop" && !called {
				t.Fatal("warning blocked command execution")
			}
		})
	}
	transactions, _ := r.LoadTransactions()
	accounts, _ := r.LoadAccounts()
	rules, _ := r.LoadRecurringRules()
	if len(transactions) != 0 || accounts[0].BalanceMinor != 10000 || rules[0].LastGeneratedMonth != "" {
		t.Fatal("startup reminder generated transactions or changed balances")
	}
	rule.LastGeneratedMonth = now.Format("2006-01")
	if err := r.SaveRecurringRules([]model.RecurringRule{rule}); err != nil {
		t.Fatal(err)
	}
	rootCmd = &cobra.Command{Use: "coinw", Run: func(cmd *cobra.Command, args []string) {}}
	rootCmd.SetArgs([]string{})
	var warnings bytes.Buffer
	rootCmd.SetErr(&warnings)
	if err := Execute(); err != nil || warnings.Len() != 0 {
		t.Fatalf("caught-up rules should not warn: %s, %v", warnings.String(), err)
	}
}

func TestRecurringWarningReadErrorDoesNotBlockCommands(t *testing.T) {
	r := setupRecurringRepository(t)
	path, err := r.DataFilePath(repository.RecurringFileName)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("invalid json"), 0o600); err != nil {
		t.Fatal(err)
	}
	previousRoot := rootCmd
	t.Cleanup(func() { rootCmd = previousRoot })
	called := false
	rootCmd = &cobra.Command{Use: "coinw", Run: func(cmd *cobra.Command, args []string) { called = true }}
	rootCmd.SetArgs([]string{})
	var warnings bytes.Buffer
	rootCmd.SetErr(&warnings)
	if err := Execute(); err != nil || !called || !strings.Contains(warnings.String(), "could not check recurring") {
		t.Fatalf("read failure blocked command or went unreported: called=%v, err=%v, stderr=%s", called, err, warnings.String())
	}
}
