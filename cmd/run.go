package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"sysup-go/internal/mise"
	"sysup-go/internal/program"
	"sysup-go/internal/runner"
	"sysup-go/internal/sudo"
)

func (a *app) runPlan(cmd *cobra.Command, _ []string) error {
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	specs, err := a.loadSpecs()
	if err != nil {
		return err
	}
	log := a.logger(os.Stderr)
	if a.logFileOut != nil {
		defer func() {
			if err := a.logFileOut.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "log-file close: %v\n", err)
			}
		}()
	}

	if a.cfg.Sudo.Keepalive {
		ka := sudo.KeepAlive{
			Interval: a.cfg.Sudo.Interval,
			Log:      log,
		}
		stopKA, err := ka.Start(ctx)
		if err != nil {
			return err
		}
		defer stopKA()
	}

	wrap := mise.Wrap{LookPath: exec.LookPath, Log: log}
	st := wrap.Begin(a.cfg.Mise.Wrap && a.cfg.Mise.StripBinsFromPath)
	childEnv := st.ChildEnv(os.Environ())

	g := a.cfg.Retries
	runnable := program.Runnable(specs)
	steps := make([]program.Runner, 0, len(runnable))
	for _, s := range runnable {
		steps = append(steps, program.Exec{
			Spec:     s,
			Stdout:   cmd.OutOrStdout(),
			Stderr:   cmd.ErrOrStderr(),
			Log:      log,
			Attempts: program.Attempts(g.Always, g.MaxAttempts, s.Retries.Always, s.Retries.MaxAttempts),
			Always:   program.Always(g.Always, s.Retries.Always),
			Env:      childEnv,
		})
	}
	planErr := runner.RunAll(ctx, log, steps, a.continueOnErr)
	restErr := st.RestoreProcessPath()
	if planErr != nil {
		return planErr
	}
	if restErr != nil {
		return restErr
	}
	return st.Up(ctx)
}
