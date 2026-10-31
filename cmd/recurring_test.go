package cmd

import (
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/amiraminb/coinwarrior/internal/service"
	"github.com/amiraminb/coinwarrior/internal/tui"
)

func setupRecurringRepository(t *testing.T) *repository.FileRepository {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	r := repository.NewFileRepository()
	if err := r.SaveCategories([]string{"Subscriptions", "Utilities", "Income"}); err != nil {
		t.Fatal(err)
	}
	if err := r.SaveAccounts([]model.Account{
		{Name: "Checking", Currency: "CAD", BalanceMinor: 10000},
		{Name: "Savings", Currency: "CAD", BalanceMinor: 5000},
	}); err != nil {
		t.Fatal(err)
	}
	previousRepo, previousService, previousTUIService := repo, svc, tui.Svc
	repo, svc = r, service.New(r)
	tui.Svc = svc
	t.Cleanup(func() { repo, svc, tui.Svc = previousRepo, previousService, previousTUIService })
	return r
}

func runRecurringCommand(t *testing.T, args ...string) string {
	t.Helper()
	return captureBudgetOutput(t, func() error {
		cmd := newRecurringCommand()
		cmd.SetArgs(args)
		return cmd.Execute()
	})
}

func TestRecurringCommandsCreateListEditAndDeleteWithoutTUI(t *testing.T) {
	r := setupRecurringRepository(t)
	accountsBefore, _ := r.LoadAccounts()
	runRecurringCommand(t, "add", "-t", "expense", "-a", "14.16", "-c", "subscriptions", "--account", " checking ", "-d", "5", "-n", "Carbon Diet", "--start-date", "2026-01-01", "--end-date", "2026-12-31")
	rules, err := r.LoadRecurringRules()
	if err != nil || len(rules) != 1 {
		t.Fatalf("rules = %+v, err = %v", rules, err)
	}
	original := rules[0]
	if original.Type != "expense" || original.AmountMinor != 1416 || original.Category != "Subscriptions" || original.Account != "Checking" || original.Currency != "CAD" || original.DayOfMonth != 5 || original.Note != "Carbon Diet" {
		t.Fatalf("unexpected rule: %+v", original)
	}
	for _, args := range [][]string{{"list"}, {"-l"}, {"--list"}} {
		output := runRecurringCommand(t, args...)
		for _, want := range []string{original.ID, "NOTE", "Carbon Diet", "14.16"} {
			if !strings.Contains(output, want) {
				t.Errorf("%v omitted %q:\n%s", args, want, output)
			}
		}
	}
	// Partial edits preserve the schedule and bookkeeping, including generation.
	original.LastGeneratedMonth = "2026-01"
	if err := r.SaveRecurringRules([]model.RecurringRule{original}); err != nil {
		t.Fatal(err)
	}
	runRecurringCommand(t, "edit", original.ID, "--amount", "19.99", "--note=", "--end-date=")
	rules, err = r.LoadRecurringRules()
	if err != nil || len(rules) != 1 {
		t.Fatalf("rules = %+v, err = %v", rules, err)
	}
	want := original
	want.AmountMinor, want.Note, want.EndDate = 1999, "", ""
	want.UpdatedAt = rules[0].UpdatedAt
	if !reflect.DeepEqual(rules[0], want) {
		t.Fatalf("partial edit = %+v, want %+v", rules[0], want)
	}
	runRecurringCommand(t, "delete", original.ID)
	rules, err = r.LoadRecurringRules()
	if err != nil || len(rules) != 0 {
		t.Fatalf("rules after delete = %+v, err = %v", rules, err)
	}
	accounts, _ := r.LoadAccounts()
	transactions, _ := r.LoadTransactions()
	if !reflect.DeepEqual(accounts, accountsBefore) || len(transactions) != 0 {
		t.Fatal("rule management changed balances or generated transactions")
	}
}

func TestRecurringGenerateAllTypesIsIdempotent(t *testing.T) {
	r := setupRecurringRepository(t)
	start := time.Now().Format("2006-01") + "-01"
	for _, fields := range [][]string{
		{"--type", "expense", "--amount", "10", "--category", "Utilities"},
		{"--type", "income", "--amount", "30", "--category", "Income"},
		{"--type", "transfer", "--amount", "5", "--to-account", "savings"},
	} {
		args := append([]string{"add", "--account", "Checking", "--day", "28", "--start-date", start}, fields...)
		runRecurringCommand(t, args...)
	}
	rules, _ := r.LoadRecurringRules()
	if rules[2].Category != model.TransferCategory || rules[2].ToAccount != "Savings" {
		t.Fatalf("unexpected transfer rule: %+v", rules[2])
	}
	output := runRecurringCommand(t, "generate")
	if !strings.Contains(output, "generated 3 transaction(s)") {
		t.Fatalf("unexpected generation output: %s", output)
	}
	if output := runRecurringCommand(t, "generate"); !strings.Contains(output, "no due transactions") {
		t.Fatalf("repeat generation: %s", output)
	}
	transactions, _ := r.LoadTransactions()
	accounts, _ := r.LoadAccounts()
	if len(transactions) != 3 || accounts[0].BalanceMinor != 11500 || accounts[1].BalanceMinor != 5500 {
		t.Fatalf("unexpected generation: %d transactions, accounts %+v", len(transactions), accounts)
	}
	// Deleting the rule keeps previously generated transactions.
	runRecurringCommand(t, "delete", rules[0].ID)
	after, _ := r.LoadTransactions()
	if !reflect.DeepEqual(transactions, after) {
		t.Fatal("deleting a rule changed generated transactions")
	}
}

func TestRecurringCommandInvalidFlagsDoNotPromptOrSave(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"add", "--type", "other"}, "invalid --type"},
		{[]string{"add", "--amount", "0"}, "--amount"},
		{[]string{"add", "--day", "29"}, "--day"},
		{[]string{"add", "--category", "missing"}, "unknown category"},
		{[]string{"add", "--account", "missing"}, "unknown account"},
		{[]string{"add", "--start-date", "2026-02-30"}, "--start-date"},
		{[]string{"add", "--start-date", "2026-05-01", "--end-date", "2026-04-30"}, "end date is before"},
		{[]string{"add", "--type", "expense", "--to-account", "Savings"}, "requires a transfer"},
		{[]string{"add", "--type", "transfer", "--account", "checking", "--to-account", "Checking"}, "must be different"},
		{[]string{"add", "--type", "expense", "--category", "Transfer"}, "unknown category"},
		{[]string{"edit", "--amount", "bad"}, "--amount"},
		{[]string{"edit", "missing", "--note", "test"}, "not found"},
		{[]string{"delete", "missing"}, "not found"},
		{[]string{"generate", "extra"}, "unknown command"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			r := setupRecurringRepository(t)
			cmd := newRecurringCommand()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
			rules, _ := r.LoadRecurringRules()
			if len(rules) != 0 {
				t.Fatal("invalid command saved rules")
			}
		})
	}
}
