package runner

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"

	"sysup-go/internal/program"
)

type fake struct {
	name string
	err  error
	ran  *atomic.Int32
}

func (f fake) Run(ctx context.Context) error {
	f.ran.Add(1)
	return f.err
}

func (f fake) Meta() program.Spec {
	return program.Spec{Name: f.name}
}

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRunAllOrderAndHardFail(t *testing.T) {
	t.Parallel()
	var a, b, c atomic.Int32
	steps := []program.Runner{
		fake{name: "a", ran: &a},
		fake{name: "b", err: errors.New("boom"), ran: &b},
		fake{name: "c", ran: &c},
	}
	err := RunAll(context.Background(), discardLog(), steps)
	var se *StepError
	if !errors.As(err, &se) {
		t.Fatalf("RunAll() error = %v, want *StepError", err)
	}
	if se.Name != "b" {
		t.Fatalf("StepError.Name = %q, want b", se.Name)
	}
	if a.Load() != 1 || b.Load() != 1 {
		t.Fatalf("ran a=%d b=%d, want 1 1", a.Load(), b.Load())
	}
	if c.Load() != 0 {
		t.Fatalf("c ran %d, want 0", c.Load())
	}
}

func TestRunAllBothOK(t *testing.T) {
	t.Parallel()
	var a, b atomic.Int32
	steps := []program.Runner{
		fake{name: "a", ran: &a},
		fake{name: "b", ran: &b},
	}
	if err := RunAll(context.Background(), discardLog(), steps); err != nil {
		t.Fatal(err)
	}
	if a.Load() != 1 || b.Load() != 1 {
		t.Fatalf("ran a=%d b=%d, want 1 1", a.Load(), b.Load())
	}
}

func TestRunAllCancelledBefore(t *testing.T) {
	t.Parallel()
	var a atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := RunAll(ctx, discardLog(), []program.Runner{fake{name: "a", ran: &a}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunAll() = %v, want context.Canceled", err)
	}
	if a.Load() != 0 {
		t.Fatalf("step ran under cancelled ctx")
	}
}

type preFake struct {
	fake
	pre *atomic.Int32
}

func (p preFake) PreRun(ctx context.Context) error {
	p.pre.Add(1)
	return nil
}

func TestRunAllPreRun(t *testing.T) {
	t.Parallel()
	var ran, pre atomic.Int32
	step := preFake{fake: fake{name: "p", ran: &ran}, pre: &pre}
	if err := RunAll(context.Background(), discardLog(), []program.Runner{step}); err != nil {
		t.Fatal(err)
	}
	if pre.Load() != 1 || ran.Load() != 1 {
		t.Fatalf("pre=%d ran=%d, want 1 1", pre.Load(), ran.Load())
	}
}
