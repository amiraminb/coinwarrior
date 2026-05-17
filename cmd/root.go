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
}

func Execute() error {
	return rootCmd.Execute()
}
