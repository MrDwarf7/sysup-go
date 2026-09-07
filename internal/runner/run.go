package runner

import (
	"context"
	"log/slog"

	"sysup-go/internal/program"
)

type metaer interface {
	Meta() program.Spec
}

// RunAll runs steps in order. The first error stops the rest.
func RunAll(ctx context.Context, log *slog.Logger, steps []program.Runner) error {
	if log == nil {
		log = slog.Default()
	}
	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := stepName(step)
		log.Info("running", "step", name)
		if p, ok := step.(program.PreRunner); ok {
			if err := p.PreRun(ctx); err != nil {
				return &StepError{Name: name, Err: err}
			}
		}
		if err := step.Run(ctx); err != nil {
			return &StepError{Name: name, Err: err}
		}
		if p, ok := step.(program.PostRunner); ok {
			if err := p.PostRun(ctx); err != nil {
				return &StepError{Name: name, Err: err}
			}
		}
	}
	return nil
}

func stepName(step program.Runner) string {
	if m, ok := step.(metaer); ok {
		return m.Meta().Name
	}
	return "step"
}
