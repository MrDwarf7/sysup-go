package tests

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"testing"
	"time"

	"sysup-go/internal/program"
)

func TestExecOptionalMissing(t *testing.T) {
	t.Parallel()
	e := program.Exec{
		Spec: program.Spec{
			Name:     "maybe",
			Optional: true,
			Command:  []string{"definitely-not-on-path-sysup"},
		},
		LookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("Run() optional missing = %v, want nil", err)
	}
}

func TestExecRequiredMissing(t *testing.T) {
	t.Parallel()
	e := program.Exec{
		Spec: program.Spec{
			Name:    "need",
			Command: []string{"definitely-not-on-path-sysup"},
		},
		LookPath: func(string) (string, error) {
			return "", errors.New("not found")
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
	}
	if err := e.Run(context.Background()); err == nil {
		t.Fatal("Run() required missing = nil, want error")
	}
}

func TestExecCancelSleep(t *testing.T) {
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep not on PATH")
	}
	ctx, cancel := context.WithCancel(context.Background())
	e := program.Exec{
		Spec: program.Spec{
			Name:    "sleep",
			Command: []string{"sleep", "30"},
		},
		LookPath: func(string) (string, error) { return sleep, nil },
		Stdout:   io.Discard,
		Stderr:   io.Discard,
	}
	done := make(chan error, 1)
	start := time.Now()
	go func() { done <- e.Run(ctx) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Run() after cancel = nil, want error")
		}
		if time.Since(start) > 5*time.Second {
			t.Fatalf("cancel took %v, want << 30s", time.Since(start))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return after cancel")
	}
}

func TestExecCtxAlreadyDone(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e := program.Exec{
		Spec: program.Spec{
			Name:    "true",
			Command: []string{"true"},
		},
		LookPath: func(string) (string, error) {
			t.Fatal("LookPath called despite cancelled ctx")
			return "", nil
		},
	}
	if err := e.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() = %v, want context.Canceled", err)
	}
}

func TestExecRetryUseAsk(t *testing.T) {
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not on PATH")
	}
	script := `for a in "$@"; do [ "$a" = --useask ] && exit 0; done
echo 'can not install conflicting packages with --noconfirm' >&2
exit 1`
	e := program.Exec{
		Spec: program.Spec{
			Name:    "aur",
			Command: []string{"sh", "-c", script, "sh"},
		},
		LookPath: func(string) (string, error) { return sh, nil },
		Stdout:   io.Discard,
		Stderr:   io.Discard,
		Attempts: 2,
	}
	if err := e.Run(context.Background()); err != nil {
		t.Fatalf("retry with --useask = %v, want nil", err)
	}
}

func TestExecNoRetryWithoutAlways(t *testing.T) {
	t.Parallel()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not on PATH")
	}
	var n int
	e := program.Exec{
		Spec: program.Spec{
			Name:    "aur",
			Command: []string{"sh", "-c", "exit 1"},
		},
		LookPath: func(string) (string, error) {
			n++
			return sh, nil
		},
		Stdout:   io.Discard,
		Stderr:   io.Discard,
		Attempts: 3,
		Always:   false,
	}
	if err := e.Run(context.Background()); err == nil {
		t.Fatal("want error")
	}
	if n != 1 {
		t.Fatalf("runs = %d, want 1 (non-retryable)", n)
	}
}
