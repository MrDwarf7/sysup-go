package cmd

import (
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List discovered programs in run order",
	Long:  "List programs from programs.toml or programs/*.toml. --skip filters the list.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		specs, err := loadSpecs()
		if err != nil {
			return err
		}
		return printList(cmd.OutOrStdout(), specs)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
