package cmd

import (
	"github.com/amiraminb/coinwarrior/internal/service"
	"github.com/amiraminb/coinwarrior/internal/repository"
	"github.com/spf13/cobra"
)

var (
	repo repository.Repository = repository.NewFileRepository()
	svc                        = service.New(repo)
)

var rootCmd = &cobra.Command{
	Use:   "coinw",
	Short: "Coinwarrior CLI",
}

func Execute() error {
	return rootCmd.Execute()
}
