package cmd

import (
	"fmt"
	"time"

	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/amiraminb/coinwarrior/internal/service"
	"github.com/amiraminb/coinwarrior/internal/tui"
	"github.com/spf13/cobra"
)

var (
	repo repository.Repository = repository.NewFileRepository()
	svc                        = service.New(repo)
)

func init() {
	tui.Svc = svc
	tui.Repo = repo
}

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "coinw",
	Short:   "Coinwarrior CLI",
	Version: version,
	// main already prints the error, so silence cobra's duplicate error line; also
	// suppress the usage dump on failure so the message isn't buried in a help wall
	// (users reach usage via --help).
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() error {
	// Check before Cobra dispatch so help and other commands also get the reminder.
	now := time.Now()
	pending, err := svc.PendingRecurringTransactionCount(now)
	if err != nil {
		fmt.Fprintf(rootCmd.ErrOrStderr(), "Warning: could not check recurring transactions: %v\n", err)
	} else if pending > 0 {
		fmt.Fprintf(rootCmd.ErrOrStderr(), "Warning: %d recurring transaction(s) have not been generated through %s. Run 'coinw recurring generate'.\n", pending, now.Format("2006-01"))
	}
	return rootCmd.Execute()
}
