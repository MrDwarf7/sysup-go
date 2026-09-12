package program

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"time"
)

const noconfirmConflict = "can not install conflicting packages with --noconfirm"

const aurRPCDrop = "connection closed before message completed"

// useAskFlag is paru's switch to pass pacman's --ask and auto-confirm
// conflicts. --noconfirm alone refuses inner conflicts; man paru:
// "--useask  Use pacman's --ask flag to automatically confirm package
// conflicts."
const useAskFlag = "--useask"

type Exec struct {
	Spec     Spec
	Stdout   io.Writer
	Stderr   io.Writer
	LookPath func(string) (string, error)
	Log      *slog.Logger
	Attempts int
	Always   bool
	Env      []string
}

func (e Exec) Meta() Spec {
	return e.Spec
}

func (e Exec) Run(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	attempts := max(e.Attempts, 1)
	argv := append([]string{}, e.Spec.Command...)
	var last error
	var captured bytes.Buffer
	for i := 1; i <= attempts; i++ {
		captured.Reset()
		last = e.runOnce(ctx, argv, &captured)
		if last == nil || !e.shouldRetry(ctx, captured.String(), last, i, attempts) {
			return last
		}
		if noconfirmConflicted(captured.String()) {
			argv = withFlag(argv, useAskFlag)
		}
		log := e.Log
		if log == nil {
			log = slog.Default()
		}
		log.WarnContext(ctx, "retrying", "step", e.Spec.Name, "attempt", i+1, "max", attempts)
	}
	return last
}

func (e Exec) shouldRetry(ctx context.Context, output string, err error, i, attempts int) bool {
	if err == nil || ctx.Err() != nil || i == attempts {
		return false
	}
	return e.Always || retryable(output, err)
}

func (e Exec) runOnce(ctx context.Context, argv []string, captured *bytes.Buffer) error {
	look := e.LookPath
	if look == nil {
		look = exec.LookPath
	}
	argv0 := argv[0]
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
	cmd := exec.CommandContext(ctx, path, argv[1:]...)
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = 3 * time.Second
	if len(e.Env) > 0 {
		cmd.Env = e.Env
	}
	cmd.Stdout = io.MultiWriter(orWriter(e.Stdout, os.Stdout), captured)
	cmd.Stderr = io.MultiWriter(orWriter(e.Stderr, os.Stderr), captured)
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func retryable(output string, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return noconfirmConflicted(output) || strings.Contains(output, aurRPCDrop)
}

func noconfirmConflicted(output string) bool {
	return strings.Contains(output, noconfirmConflict)
}

func withFlag(argv []string, flag string) []string {
	if slices.Contains(argv, flag) {
		return argv
	}
	out := make([]string, len(argv)+1)
	copy(out, argv)
	out[len(argv)] = flag
	return out
}

func orWriter(w, fallback io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return fallback
}
