package program

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
)

type Exec struct {
	Spec     Spec
	Stdout   io.Writer
	Stderr   io.Writer
	LookPath func(string) (string, error)
	Log      *slog.Logger
}

func (e Exec) Meta() Spec {
	return e.Spec
}

func (e Exec) Run(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(e.Spec.Command) < 1 {
		return fmt.Errorf("command required")
	}

	look := e.LookPath
	if look == nil {
		look = exec.LookPath
	}

	argv0 := e.Spec.Command[0]
	path, err := look(argv0)
	if err != nil && e.Spec.Optional {
		log := e.Log
		if log == nil {
			log = slog.Default()
		}
		log.InfoContext(ctx, "skipping missing optional program", "name", e.Spec.Name, "argv0", argv0)
		return nil
	}
	if err != nil {
		return fmt.Errorf("lookpath %s: %w", argv0, err)
	}

	cmd := exec.CommandContext(ctx, path, e.Spec.Command[1:]...)
	cmd.Stdout = orWriter(e.Stdout, os.Stdout)
	cmd.Stderr = orWriter(e.Stderr, os.Stderr)
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func orWriter(w, fallback io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return fallback
}
