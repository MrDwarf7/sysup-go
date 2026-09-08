package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"sysup-go/internal/config"
	"sysup-go/internal/resolve"
)

func (a *app) validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Parse config and programs without running them",
		Long:  "Load config, discover programs, resolve PKG_MANAGER, and apply --skip. Does not exec recipes.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			specs, err := a.loadSpecs()
			if err != nil {
				return err
			}
			r := resolve.Resolver{
				LookPath:  exec.LookPath,
				LookupEnv: os.LookupEnv,
			}
			pkgPath, err := r.PkgManager(cmd.Context(), a.cfg.PkgManager.Name)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			label := a.origin.Kind.String()
			if a.origin.Kind == config.KindFile {
				label = a.origin.Path
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
}
