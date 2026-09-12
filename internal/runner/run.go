package runner

import (
	"context"
	"errors"
	"log/slog"

	"sysup-go/internal/program"
)

func RunAll(ctx context.Context, log *slog.Logger, steps []program.Runner, continueOnErr bool) error {
	if log == nil {
		log = slog.Default()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var collected []StepError
	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			if len(collected) > 0 {
				return &ContinueError{Steps: collected}
			}
			return err
		}
		name := step.Meta().Name
		log.InfoContext(ctx, "running", "step", name)
		if err := runOne(ctx, step, name); err != nil {
			se := stepError(name, err)
			if !continueOnErr {
				return se
			}
			log.ErrorContext(ctx, "step failed", "step", name, "err", se.Err)
			collected = append(collected, *se)
		}
	}
	if len(collected) > 0 {
		return &ContinueError{Steps: collected}
	}
	return nil
}

func stepError(name string, err error) *StepError {
	if se, ok := errors.AsType[*StepError](err); ok {
		return se
	}
	return &StepError{Name: name, Err: err}
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
