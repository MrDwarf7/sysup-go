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
	// Capture only when a later attempt may need child text AND we are
	// not on a real *os.File display. Pipes/MultiWriter/PTY all break
	// the real TTY: sudo can no longer hide passwords, and capture
	// buffers would retain secrets. Live TTY runs pass files through
	// and classify retries without reading the stream.
	needCapture := attempts > 1
	argv := append([]string{}, e.Spec.Command...)
	var last error
	var captured bytes.Buffer
	for i := 1; i <= attempts; i++ {
		var buf *bytes.Buffer
		if needCapture {
			captured.Reset()
			buf = &captured
		}
		last = e.runOnce(ctx, argv, buf)
		out := bufString(buf)
		if last == nil || !e.shouldRetry(ctx, out, last, i, attempts, argv) {
			return last
		}
		argv = e.prepareRetry(argv, out)
		e.logRetry(ctx, i+1, attempts)
	}
	return last
}

func bufString(buf *bytes.Buffer) string {
	if buf == nil {
		return ""
	}
	return buf.String()
}

func (e Exec) prepareRetry(argv []string, out string) []string {
	if noconfirmConflicted(out) || (out == "" && hasFlag(argv, "--noconfirm")) {
		return withFlag(argv, useAskFlag)
	}
	return argv
}

func (e Exec) logRetry(ctx context.Context, attempt, maxAttempts int) {
	log := e.Log
	if log == nil {
		log = slog.Default()
	}
	log.WarnContext(ctx, "retrying", "step", e.Spec.Name, "attempt", attempt, "max", maxAttempts)
}

func (e Exec) shouldRetry(ctx context.Context, output string, err error, i, attempts int, argv []string) bool {
	if err == nil || ctx.Err() != nil || i == attempts {
		return false
	}
	if e.Always || retryable(output, err) {
		return true
	}
	// Live TTY path skips capture (output ""). Still escalate a
	// --noconfirm failure with --useask: that is the AUR conflict case
	// the recipe max_attempts is for.
	return output == "" && hasFlag(argv, "--noconfirm") && !hasFlag(argv, useAskFlag)
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
	stdout := orWriter(e.Stdout, os.Stdout)
	stderr := orWriter(e.Stderr, os.Stderr)
	cmd := e.command(ctx, path, argv[1:])

	// Real *os.File (interactive terminal or a test PTY slave): give
	// the child the fds directly. Stdin is always the process TTY so
	// sudo/pacman own echo and password entry. Never tee or PTY-wrap
	// this path -- that is what printed passwords in the clear.
	if fileWriter(stdout) || fileWriter(stderr) {
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		cmd.Stdin = os.Stdin
		if captured != nil {
			captured.Reset() // no bytes; callers treat empty as live-TTY
		}
		return cmd.Run()
	}

	if captured != nil {
		cmd.Stdout = io.MultiWriter(stdout, captured)
		cmd.Stderr = io.MultiWriter(stderr, captured)
	} else {
		cmd.Stdout = stdout
		cmd.Stderr = stderr
	}
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (e Exec) command(ctx context.Context, path string, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, path, args...)
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
	return cmd
}

func fileWriter(w io.Writer) bool {
	_, ok := w.(*os.File)
	return ok
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

func hasFlag(argv []string, flag string) bool {
	return slices.Contains(argv, flag)
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
