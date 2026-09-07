package program

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
)

// Exec runs Spec.Command via exec.CommandContext. It implements Runner only.
type Exec struct {
	Spec     Spec
	Stdout   io.Writer
	Stderr   io.Writer
	LookPath func(string) (string, error)
	Log      *slog.Logger
}

// Meta returns the underlying spec.
func (e Exec) Meta() Spec {
	return e.Spec
}

// Run starts the command bound to ctx.
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
	if err != nil {
		if e.Spec.Optional {
			log := e.Log
			if log == nil {
				log = slog.Default()
			}
			log.Info("skipping missing optional program", "name", e.Spec.Name, "argv0", argv0)
			return nil
		}
		return fmt.Errorf("lookpath %s: %w", argv0, err)
	}
	cmd := exec.CommandContext(ctx, path, e.Spec.Command[1:]...)
	stdout := e.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := e.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
