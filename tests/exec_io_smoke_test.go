package tests

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"

	"sysup-go/internal/program"
)

// mustPTY opens a PTY slave suitable as Exec Stdout/Stderr. Skips on
// hosts without PTY support (Windows CI, some containers).
func mustPTY(t *testing.T) (tty *os.File) {
	t.Helper()
	m, s, err := pty.Open()
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = m.Close()
		_ = s.Close()
	})
	go func() { _, _ = io.Copy(io.Discard, m) }()
	return s
}

func TestExecSingleAttemptChildSeesTTY(t *testing.T) {
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not on PATH")
	}
	tty := mustPTY(t)

	e := program.Exec{
		Spec:     program.Spec{Name: "tty1", Command: []string{"sh", "-c", ttyProbeScript()}},
		LookPath: func(string) (string, error) { return sh, nil },
		Stdout:   tty,
		Stderr:   tty,
		Attempts: 1,
	}
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("single-attempt child should see TTY: %v", err)
	}
}

func TestExecMultiAttemptChildSeesTTY(t *testing.T) {
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not on PATH")
	}
	tty := mustPTY(t)

	// Multi-attempt still passes *os.File through (no PTY wrap) so the
	// child keeps isatty and sudo can own the real stdin TTY.
	e := program.Exec{
		Spec:     program.Spec{Name: "tty2", Command: []string{"sh", "-c", ttyProbeScript()}},
		LookPath: func(string) (string, error) { return sh, nil },
		Stdout:   tty,
		Stderr:   tty,
		Attempts: 2,
	}
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("multi-attempt child should see TTY: %v", err)
	}
}

// TestExecStreamingNotBatched checks flushed child lines arrive as
// separate writes over time. Relative spread, not absolute latency.
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
for i in range(4):
    sys.stdout.write("line-%d\n" % i)
    sys.stdout.flush()
    time.sleep(0.2)
'`
	tw := &chunkWriter{}
	e := program.Exec{
		Spec:     program.Spec{Name: "stream", Command: []string{"sh", "-c", script}},
		LookPath: func(string) (string, error) { return sh, nil },
		Stdout:   tw,
		Stderr:   io.Discard,
		Attempts: 1,
	}
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("stream run: %v", err)
	}
	times := tw.times()
	if len(times) < 2 {
		t.Fatalf("got %d write(s), want >= 2 separate chunks (batched?)", len(times))
	}
	spread := times[len(times)-1].Sub(times[0])
	if spread < 150*time.Millisecond {
		t.Fatalf("write spread %v too small: output arrived as one burst", spread)
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

// TestExecTTYNoconfirmRetriesWithoutCapture: live *os.File display
// skips stream capture (sudo/password safety). --noconfirm failures
// still escalate with --useask via the empty-output heuristic.
func TestExecTTYNoconfirmRetriesWithoutCapture(t *testing.T) {
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not on PATH")
	}
	tty := mustPTY(t)

	script := `for a in "$@"; do [ "$a" = --useask ] && exit 0; done
echo 'can not install conflicting packages with --noconfirm' >&2
exit 1`
	e := program.Exec{
		Spec: program.Spec{
			Name:    "aur-tty",
			Command: []string{"sh", "-c", script, "sh", "--noconfirm"},
		},
		LookPath: func(string) (string, error) { return sh, nil },
		Stdout:   tty,
		Stderr:   tty,
		Attempts: 2,
	}
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("TTY noconfirm retry with --useask = %v, want nil", err)
	}
}

// TestExecStdinIsRealFile ensures children inherit a readable stdin
// fd (the process TTY in interactive runs). sudo relies on that to
// prompt and hide passwords.
func TestExecStdinIsRealFile(t *testing.T) {
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not on PATH")
	}
	e := program.Exec{
		Spec:     program.Spec{Name: "stdin", Command: []string{"sh", "-c", "test -r /dev/fd/0 || test -r /dev/stdin"}},
		LookPath: func(string) (string, error) { return sh, nil },
		Stdout:   io.Discard,
		Stderr:   io.Discard,
		Attempts: 1,
	}
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("child should inherit readable stdin: %v", err)
	}
}

func ttyProbeScript() string {
	return `test -t 1`
}

type chunkWriter struct {
	mu     sync.Mutex
	stamps []time.Time
}

func (c *chunkWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	c.mu.Lock()
	c.stamps = append(c.stamps, time.Now())
	c.mu.Unlock()
	return len(p), nil
}

func (c *chunkWriter) times() []time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]time.Time, len(c.stamps))
	copy(out, c.stamps)
	return out
}
