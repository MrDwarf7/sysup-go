package tests

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"sysup-go/internal/program"
	"sysup-go/internal/runner"
)

// fake is a sequential Runner. ran is *atomic.Int32 (shared cell) so
// value-typed fakes can still bump a counter without *int jank.
type fake struct {
	name string
	err  error
	ran  *atomic.Int32
}

func (f fake) Run(_ context.Context) error {
	if f.ran != nil {
		f.ran.Add(1)
	}
	return f.err
}

func (f fake) Meta() program.Spec {
	return program.Spec{Name: f.name}
}

type preFake struct {
	fake
	pre *atomic.Int32
}

func (p preFake) PreRun(_ context.Context) error {
	if p.pre != nil {
		p.pre.Add(1)
	}
	return nil
}

func TestRunAllOrderAndHardFail(t *testing.T) {
	t.Parallel()
	var a, b, c atomic.Int32
	steps := []program.Runner{
		fake{name: "a", ran: &a},
		fake{name: "b", err: errors.New("boom"), ran: &b},
		fake{name: "c", ran: &c},
	}
	err := runner.RunAll(context.Background(), discardLog(), steps, false)
	var se *runner.StepError
	if !errors.As(err, &se) {
		t.Fatalf("RunAll() error = %v, want *runner.StepError", err)
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
	if err := runner.RunAll(context.Background(), discardLog(), steps, false); err != nil {
		t.Fatal(err)
	}
	if a.Load() != 1 || b.Load() != 1 {
		t.Fatalf("ran a=%d b=%d, want 1 1", a.Load(), b.Load())
	}
}

func TestRunAllCancelledBefore(t *testing.T) {
	t.Parallel()
	var a atomic.Int32
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := runner.RunAll(ctx, discardLog(), []program.Runner{fake{name: "a", ran: &a}}, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunAll() = %v, want context.Canceled", err)
	}
	if a.Load() != 0 {
		t.Fatalf("step ran under cancelled ctx")
	}
}

func TestRunAllPreRun(t *testing.T) {
	t.Parallel()
	var ran, pre atomic.Int32
	step := preFake{fake{name: "p", ran: &ran}, &pre}
	if err := runner.RunAll(context.Background(), discardLog(), []program.Runner{step}, false); err != nil {
		t.Fatal(err)
	}
	if pre.Load() != 1 || ran.Load() != 1 {
		t.Fatalf("pre=%d ran=%d, want 1 1", pre.Load(), ran.Load())
	}
}

func TestRunAllContinue(t *testing.T) {
	t.Parallel()
	var a, b, c atomic.Int32
	steps := []program.Runner{
		fake{name: "a", ran: &a},
		fake{name: "b", err: errors.New("boom"), ran: &b},
		fake{name: "c", ran: &c},
	}
	err := runner.RunAll(context.Background(), discardLog(), steps, true)
	var es *runner.ContinueError
	if !errors.As(err, &es) {
		t.Fatalf("RunAll() error = %v, want *runner.ContinueError", err)
	}
	if len(es.Steps) != 1 || es.Steps[0].Name != "b" {
		t.Fatalf("ContinueError = %+v, want one step b", es.Steps)
	}
	if a.Load() != 1 || b.Load() != 1 || c.Load() != 1 {
		t.Fatalf("ran a=%d b=%d c=%d, want 1 1 1", a.Load(), b.Load(), c.Load())
	}
}

func TestRunAllContinueTwoFailures(t *testing.T) {
	t.Parallel()
	var a, b, c atomic.Int32
	steps := []program.Runner{
		fake{name: "a", err: errors.New("a"), ran: &a},
		fake{name: "b", ran: &b},
		fake{name: "c", err: errors.New("c"), ran: &c},
	}
	err := runner.RunAll(context.Background(), discardLog(), steps, true)
	var es *runner.ContinueError
	if !errors.As(err, &es) {
		t.Fatalf("RunAll() error = %v, want *runner.ContinueError", err)
	}
	if len(es.Steps) != 2 || es.Steps[0].Name != "a" || es.Steps[1].Name != "c" {
		t.Fatalf("ContinueError = %+v, want a then c", es.Steps)
	}
	if a.Load() != 1 || b.Load() != 1 || c.Load() != 1 {
		t.Fatalf("ran a=%d b=%d c=%d, want 1 1 1", a.Load(), b.Load(), c.Load())
	}
}

func TestRunAllContinueAllOK(t *testing.T) {
	t.Parallel()
	var a atomic.Int32
	if err := runner.RunAll(context.Background(), discardLog(), []program.Runner{fake{name: "a", ran: &a}}, true); err != nil {
		t.Fatal(err)
	}
	if a.Load() != 1 {
		t.Fatalf("ran a=%d, want 1", a.Load())
	}
}

func TestRunAllContinueCancelAfterFirst(t *testing.T) {
	t.Parallel()
	var a, b, c atomic.Int32
	ctx, cancel := context.WithCancel(t.Context())
	steps := []program.Runner{
		fake{name: "a", ran: &a},
		cancelFake{fake{name: "b", ran: &b}, cancel},
		fake{name: "c", ran: &c},
	}
	err := runner.RunAll(ctx, discardLog(), steps, true)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunAll() = %v, want context.Canceled", err)
	}
	if a.Load() != 1 || b.Load() != 1 {
		t.Fatalf("ran a=%d b=%d, want 1 1", a.Load(), b.Load())
	}
}

type cancelFake struct {
	fake
	cancel context.CancelFunc
}

func (c cancelFake) Run(ctx context.Context) error {
	if err := c.fake.Run(ctx); err != nil {
		return err
	}
	c.cancel()
	return nil
}
