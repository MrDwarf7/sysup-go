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
			Command:  []string{"definitely-not-on-path-sysup-go"},
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
			Command: []string{"definitely-not-on-path-sysup-go"},
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
