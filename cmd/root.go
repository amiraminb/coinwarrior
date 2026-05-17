package cmd

import (
	coininternal "github.com/amiraminb/coinwarrior/internal"
	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/spf13/cobra"
)

var (
	repo repository.Repository = repository.NewFileRepository()
	svc                        = coininternal.NewService(repo)
)

var rootCmd = &cobra.Command{
	Use:   "coinw",
	Short: "Coinwarrior CLI",
}

func Execute() error {
	return rootCmd.Execute()
}
