package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"sysup-go/internal/program"
	"sysup-go/internal/runner"
)

func (a *app) runPlan(cmd *cobra.Command, _ []string) error {
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	specs, err := a.loadSpecs()
	if err != nil {
		return err
	}
	log := a.logger()
	steps := make([]program.Runner, 0, len(specs))
	for _, s := range specs {
		steps = append(steps, program.Exec{
			Spec:   s,
			Stdout: cmd.OutOrStdout(),
			Stderr: cmd.ErrOrStderr(),
			Log:    log,
		})
	}
	return runner.RunAll(ctx, log, steps)
}
