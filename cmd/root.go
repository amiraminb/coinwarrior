package cmd

import (
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
	return rootCmd.Execute()
}
