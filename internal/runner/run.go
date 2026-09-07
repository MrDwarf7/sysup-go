package runner

import (
	"context"
	"log/slog"

	"sysup-go/internal/program"
)

func RunAll(ctx context.Context, log *slog.Logger, steps []program.Runner) error {
	if log == nil {
		log = slog.Default()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, step := range steps {
		name := step.Meta().Name
		log.Info("running", "step", name)
		if err := runOne(ctx, step, name); err != nil {
			return err
		}
	}
	return nil
}

func runOne(ctx context.Context, step program.Runner, name string) error {
	if err := maybePre(ctx, step); err != nil {
		return &StepError{Name: name, Err: err}
	}
	if err := step.Run(ctx); err != nil {
		return &StepError{Name: name, Err: err}
	}
	if err := maybePost(ctx, step); err != nil {
		return &StepError{Name: name, Err: err}
	}
	return nil
}

func maybePre(ctx context.Context, step program.Runner) error {
	hook, ok := step.(program.PreRunner)
	if !ok {
		return nil
	}
	return hook.PreRun(ctx)
}

func maybePost(ctx context.Context, step program.Runner) error {
	hook, ok := step.(program.PostRunner)
	if !ok {
		return nil
	}
	return hook.PostRun(ctx)
}
