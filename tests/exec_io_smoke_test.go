package tests

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"testing"
	"time"

	"github.com/creack/pty"

	"sysup-go/internal/program"
)

func TestExecSingleAttemptChildSeesTTY(t *testing.T) {
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not on PATH")
	}
	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open: %v", err)
	}
	defer func() { _ = ptmx.Close() }()
	defer func() { _ = tty.Close() }()
	go func() { _, _ = io.Copy(io.Discard, ptmx) }()

	script := ttyProbeScript()
	e := program.Exec{
		Spec:     program.Spec{Name: "tty1", Command: []string{"sh", "-c", script}},
		LookPath: func(string) (string, error) { return sh, nil },
		Stdout:   tty,
		Stderr:   tty,
		Attempts: 1,
	}
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("single-attempt child should see TTY: %v", err)
	}
}

func TestExecMultiAttemptPTYChildSeesTTY(t *testing.T) {
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not on PATH")
	}
	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open: %v", err)
	}
	defer func() { _ = ptmx.Close() }()
	defer func() { _ = tty.Close() }()
	go func() { _, _ = io.Copy(io.Discard, ptmx) }()

	script := ttyProbeScript()
	e := program.Exec{
		Spec:     program.Spec{Name: "tty2", Command: []string{"sh", "-c", script}},
		LookPath: func(string) (string, error) { return sh, nil },
		Stdout:   tty,
		Stderr:   tty,
		Attempts: 2,
	}
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("multi-attempt PTY child should see TTY: %v", err)
	}
}

func TestExecStreamingNotBatched(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not on PATH")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not on PATH")
	}
	script := `python3 -c '
import sys, time
for i in range(5):
    print(f"line-{i}", flush=True)
    time.sleep(0.15)
'`
	tw := &timedWriter{}
	e := program.Exec{
		Spec:     program.Spec{Name: "stream", Command: []string{"sh", "-c", script}},
		LookPath: func(string) (string, error) { return sh, nil },
		Stdout:   tw,
		Stderr:   io.Discard,
		Attempts: 1,
	}
	start := time.Now()
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("stream run: %v", err)
	}
	if tw.first.IsZero() {
		t.Fatal("no bytes received")
	}
	firstAt := tw.first.Sub(start)
	total := time.Since(start)
	if firstAt > 300*time.Millisecond {
		t.Fatalf("first byte at %v (total %v): output still batched", firstAt, total)
	}
	if total < 500*time.Millisecond {
		t.Fatalf("finished too fast (%v): delays not applied", total)
	}
}

func TestExecRetryCapturePipePath(t *testing.T) {
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not on PATH")
	}
	script := `for a in "$@"; do [ "$a" = --useask ] && exit 0; done
echo 'can not install conflicting packages with --noconfirm' >&2
exit 1`
	var sink bytes.Buffer
	e := program.Exec{
		Spec:     program.Spec{Name: "retry", Command: []string{"sh", "-c", script, "sh"}},
		LookPath: func(string) (string, error) { return sh, nil },
		Stdout:   &sink,
		Stderr:   &sink,
		Attempts: 2,
	}
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("retry with --useask = %v, want nil", err)
	}
}

// TestExecPTYCapturesRetryText ensures the PTY tee still feeds the
// capture buffer so classified retries keep working on a TTY display.
func TestExecPTYCapturesRetryText(t *testing.T) {
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not on PATH")
	}
	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open: %v", err)
	}
	defer func() { _ = ptmx.Close() }()
	defer func() { _ = tty.Close() }()
	go func() { _, _ = io.Copy(io.Discard, ptmx) }()

	script := `for a in "$@"; do [ "$a" = --useask ] && exit 0; done
echo 'can not install conflicting packages with --noconfirm' >&2
exit 1`
	e := program.Exec{
		Spec:     program.Spec{Name: "aur-pty", Command: []string{"sh", "-c", script, "sh"}},
		LookPath: func(string) (string, error) { return sh, nil },
		Stdout:   tty,
		Stderr:   tty,
		Attempts: 2,
	}
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("PTY retry with --useask = %v, want nil", err)
	}
}

func ttyProbeScript() string {
	return `if command -v python3 >/dev/null; then python3 -c 'import os,sys; sys.exit(0 if os.isatty(1) else 1)'; else test -t 1; fi`
}

type timedWriter struct {
	first time.Time
}

func (t *timedWriter) Write(p []byte) (int, error) {
	if t.first.IsZero() && len(p) > 0 {
		t.first = time.Now()
	}
	return len(p), nil
}
