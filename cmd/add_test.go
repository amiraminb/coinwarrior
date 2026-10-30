package cmd

import (
	"io"
	"strings"
	"testing"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/amiraminb/coinwarrior/internal/service"
	"github.com/amiraminb/coinwarrior/internal/tui"
)

func setupAddRepository(t *testing.T) *repository.FileRepository {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	r := repository.NewFileRepository()
	if err := r.SaveAccounts([]model.Account{{Name: "Checking", Currency: "CAD", BalanceMinor: 10000}}); err != nil {
		t.Fatal(err)
	}
	previous := tui.Svc
	tui.Svc = service.New(r)
	t.Cleanup(func() { tui.Svc = previous })
	return r
}

func TestAddCommandSavesFromFlags(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		txType  string
		note    string
		balance int64
	}{
		{
			"short flags and empty note",
			[]string{"-t", "expense", "-a", "20", "-d", "2026-05-21", "-c", "housing", "--account", "checking", "-n", ""},
			"expense", "", 8000,
		},
		{
			"long flags and note",
			[]string{"--type", "expense", "--amount", "20", "--date", "2026-05-21", "--category", "Housing", "--account", "Checking", "--note", "monthly rent"},
			"expense", "monthly rent", 8000,
		},
		{
			"income",
			[]string{"-t", "income", "-a", "20", "-d", "2026-05-21", "-c", "Housing", "--account", "Checking", "--note="},
			"income", "", 12000,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := setupAddRepository(t)
			cmd := newAddCommand()
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("fully specified command should save without a TUI: %v", err)
			}
			transactions, err := r.LoadTransactions()
			if err != nil {
				t.Fatal(err)
			}
			if len(transactions) != 1 {
				t.Fatalf("saved %d transactions, want 1", len(transactions))
			}
			tx := transactions[0]
			if tx.Type != tc.txType || tx.AmountMinor != 2000 || tx.Date != "2026-05-21" ||
				tx.Category != "Housing" || tx.Account != "Checking" || tx.Currency != "CAD" || tx.Note != tc.note {
				t.Errorf("saved unexpected transaction: %+v", tx)
			}
			accounts, err := r.LoadAccounts()
			if err != nil {
				t.Fatal(err)
			}
			if accounts[0].BalanceMinor != tc.balance {
				t.Errorf("balance = %d, want %d", accounts[0].BalanceMinor, tc.balance)
			}
		})
	}
}

func TestAddCommandRejectsInvalidFlagsBeforePrompting(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"-a", "bad"}, "--amount"},
		{[]string{"-d", "2026-02-30"}, "--date"},
		{[]string{"-t", "unknown"}, "--type"},
		{[]string{"-c", ""}, "--category"},
		{[]string{"--account", "Missing"}, "unknown account"},
		{[]string{"unexpected"}, "unknown command"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			r := setupAddRepository(t)
			cmd := newAddCommand()
			cmd.SetArgs(tc.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got error %v, want %q", err, tc.want)
			}
			transactions, err := r.LoadTransactions()
			if err != nil {
				t.Fatal(err)
			}
			if len(transactions) != 0 {
				t.Errorf("invalid flags saved %d transactions", len(transactions))
			}
		})
	}
}
