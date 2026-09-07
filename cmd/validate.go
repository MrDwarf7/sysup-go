package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"sysup-go/internal/config"
	"sysup-go/internal/resolve"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Parse config and programs without running them",
	Long:  "Load config, discover programs, resolve PKG_MANAGER, and apply --skip. Does not exec recipes.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Unmarshal(viper.GetViper())
		if err != nil {
			return err
		}
		specs, err := loadSpecs()
		if err != nil {
			return err
		}
		r := resolve.Resolver{
			LookPath:  exec.LookPath,
			LookupEnv: os.LookupEnv,
		}
		pkgPath, err := r.PkgManager(cmd.Context(), cfg.PkgManager.Name)
		if err != nil {
			return err
		}
		out := cmd.OutOrStdout()
		origin := config.OriginFrom(viper.GetViper())
		label := origin.Kind.String()
		if origin.Kind == config.KindFile {
			label = origin.Path
		}
		if _, err := fmt.Fprintf(out, "config:  %s\n", label); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "pkg_manager: %s\n", pkgPath); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "programs: %d\n", len(specs)); err != nil {
			return err
		}
		return printList(out, specs)
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
