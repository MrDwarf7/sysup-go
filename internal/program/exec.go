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

	"github.com/creack/pty"
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
	// Capture only when a later attempt may need the child's text
	// (classified retries / --useask). MultiWriter pipes steal the
	// TTY: full-block buffering and no colors. Skip capture on the
	// common single-attempt path so os.Stdout/os.Stderr stay real.
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
		out := ""
		if buf != nil {
			out = buf.String()
		}
		if last == nil || !e.shouldRetry(ctx, out, last, i, attempts) {
			return last
		}
		if noconfirmConflicted(out) {
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
	stdout := orWriter(e.Stdout, os.Stdout)
	stderr := orWriter(e.Stderr, os.Stderr)

	if captured == nil {
		// Real files stay files: child keeps isatty, line buffering, colors.
		cmd := e.command(ctx, path, argv[1:])
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}

	// Need the bytes for retry classification. A pipe makes the child
	// full-buffer and drop ANSI; give it a PTY when the display is a
	// terminal so streaming and colors match a direct TTY run.
	if display, ok := ttyDisplay(stdout, stderr); ok {
		err := runPTY(e.command(ctx, path, argv[1:]), display, captured)
		if err == nil || !errors.Is(err, errPTYStart) {
			return err
		}
		// PTY unavailable (chroot, missing /dev/ptmx, etc.): fall through
		// to pipes so retries still see the output text.
	}

	cmd := e.command(ctx, path, argv[1:])
	cmd.Stdout = io.MultiWriter(stdout, captured)
	cmd.Stderr = io.MultiWriter(stderr, captured)
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

// errPTYStart marks pty.Start failure so runOnce can fall back to pipes.
var errPTYStart = errors.New("pty start")

// runPTY starts cmd on a pseudo-terminal, tees the slave output to
// display and captured, and copies the real stdin into the PTY so
// sudo/password prompts still work.
func runPTY(cmd *exec.Cmd, display *os.File, captured *bytes.Buffer) error {
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("%w: %w", errPTYStart, err)
	}
	defer func() { _ = ptmx.Close() }()

	if isTTY(os.Stdin) {
		_ = pty.InheritSize(os.Stdin, ptmx)
		go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	} else {
		_ = pty.InheritSize(display, ptmx)
	}
	out := &ptyTee{display: display, buf: captured}
	copyDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(out, ptmx)
		close(copyDone)
	}()

	waitErr := cmd.Wait()
	// Unblock the copy if the child exited without closing output.
	_ = ptmx.Close()
	<-copyDone
	return waitErr
}

// ttyDisplay picks a terminal file to show PTY output on. Writing the
// merged PTY stream to both stdout and stderr would double every line
// when both are the same interactive terminal.
func ttyDisplay(stdout, stderr io.Writer) (*os.File, bool) {
	if f, ok := stdout.(*os.File); ok && isTTY(f) {
		return f, true
	}
	if f, ok := stderr.(*os.File); ok && isTTY(f) {
		return f, true
	}
	return nil, false
}

func isTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// ptyTee always records into buf. Display writes are best-effort so a
// short write or EIO on the terminal cannot starve retry capture.
type ptyTee struct {
	display io.Writer
	buf     *bytes.Buffer
}

func (t *ptyTee) Write(p []byte) (int, error) {
	if t.buf != nil {
		_, _ = t.buf.Write(p)
	}
	if t.display != nil {
		_, _ = t.display.Write(p)
	}
	return len(p), nil
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
