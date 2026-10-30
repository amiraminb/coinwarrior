package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var categoryCmd = newCategoryCommand()

func newCategoryCommand() *cobra.Command {
	list := func(cmd *cobra.Command, args []string) error {
		categories, err := svc.LoadCategories()
		if err != nil {
			return err
		}
		slices.Sort(categories)
		if len(categories) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no categories")
		}
		for _, category := range categories {
			fmt.Fprintln(cmd.OutOrStdout(), category)
		}
		return nil
	}
	cmd := &cobra.Command{
		Use: "category", Aliases: []string{"categories"},
		Short: "List and manage categories", Args: cobra.NoArgs, RunE: list,
	}
	cmd.AddCommand(&cobra.Command{Use: "list", Short: "List saved categories", Args: cobra.NoArgs, RunE: list})
	cmd.AddCommand(&cobra.Command{
		Use: "add <name>", Short: "Add a category", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("category cannot be empty")
			}
			categories, err := svc.LoadCategories()
			if err != nil {
				return err
			}
			if i := slices.IndexFunc(categories, func(c string) bool { return strings.EqualFold(c, name) }); i >= 0 {
				return fmt.Errorf("category '%s' already exists", categories[i])
			}
			if err := svc.AddCategory(name); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "category added:", name)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use: "rename <name> <new-name>", Aliases: []string{"edit"},
		Short: "Rename a category and its transactions, recurring rules, and budgets",
		Args:  cobra.ExactArgs(2), ValidArgsFunction: completeCategoryName,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := svc.RenameCategory(args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "category renamed: %s -> %s\n", strings.TrimSpace(args[0]), strings.TrimSpace(args[1]))
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use: "delete <name>", Short: "Delete a category that has no transactions, recurring rules, or budgets",
		Args: cobra.ExactArgs(1), ValidArgsFunction: completeCategoryName,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := svc.DeleteCategory(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "category deleted:", strings.TrimSpace(args[0]))
			return nil
		},
	})
	return cmd
}

func completeCategoryName(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	categories, err := svc.LoadCategories()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return categories, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(categoryCmd)
}
