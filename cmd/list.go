package cmd

import (
	"github.com/spf13/cobra"
)

func (a *app) listCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List discovered programs in run order",
		Long:    "List programs from programs.toml or programs/*.toml. --skip filters the list.",
		Aliases: []string{"ls", "show"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			specs, err := a.loadSpecs()
			if err != nil {
				return err
			}
			return printList(cmd.OutOrStdout(), specs)
		},
	}
}
