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
	// Drain master so slave writes from the parent tee never block.
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

func TestExecMultiAttemptPTYChildSeesTTY(t *testing.T) {
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not on PATH")
	}
	tty := mustPTY(t)

	e := program.Exec{
		Spec:     program.Spec{Name: "tty2", Command: []string{"sh", "-c", ttyProbeScript()}},
		LookPath: func(string) (string, error) { return sh, nil },
		Stdout:   tty,
		Stderr:   tty,
		Attempts: 2,
	}
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("multi-attempt PTY child should see TTY: %v", err)
	}
}

// TestExecStreamingNotBatched checks that flushed child lines reach the
// parent as separate writes over time, not one dump at exit. Timing is
// relative (spread across writes), not absolute wall-clock, so slow CI
// hosts and cold interpreters do not flake.
func TestExecStreamingNotBatched(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not on PATH")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not on PATH")
	}
	// 4 lines, 200ms apart, each flushed. Pipe path still streams when
	// the child flushes; we only assert multiple timed writes.
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

// TestExecPTYCapturesRetryText ensures the PTY tee still feeds the
// capture buffer so classified retries keep working when the display
// is a terminal file. Skips when the host cannot open a PTY.
func TestExecPTYCapturesRetryText(t *testing.T) {
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
	// Prefer test(1): always present with sh. python3 isatty is fine too
	// but adds a dependency some CI images lack.
	return `test -t 1`
}

// chunkWriter records the wall time of each non-empty Write so tests
// can tell streaming (many spaced writes) from a single end dump.
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
