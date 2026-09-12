package tests

import (
	"context"
	"errors"
	"testing"

	"sysup-go/internal/program"
	"sysup-go/internal/runner"
)

type fake struct {
	name string
	err  error
	ran  *int
}

func (f fake) Run(_ context.Context) error {
	*f.ran++
	return f.err
}

func (f fake) Meta() program.Spec {
	return program.Spec{Name: f.name}
}

type preFake struct {
	fake
	pre *int
}

func (p preFake) PreRun(_ context.Context) error {
	*p.pre++
	return nil
}

func TestRunAllOrderAndHardFail(t *testing.T) {
	t.Parallel()
	var a, b, c int
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
	if a != 1 || b != 1 {
		t.Fatalf("ran a=%d b=%d, want 1 1", a, b)
	}
	if c != 0 {
		t.Fatalf("c ran %d, want 0", c)
	}
}

func TestRunAllBothOK(t *testing.T) {
	t.Parallel()
	var a, b int
	steps := []program.Runner{
		fake{name: "a", ran: &a},
		fake{name: "b", ran: &b},
	}
	if err := runner.RunAll(context.Background(), discardLog(), steps, false); err != nil {
		t.Fatal(err)
	}
	if a != 1 || b != 1 {
		t.Fatalf("ran a=%d b=%d, want 1 1", a, b)
	}
}

func TestRunAllCancelledBefore(t *testing.T) {
	t.Parallel()
	var a int
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := runner.RunAll(ctx, discardLog(), []program.Runner{fake{name: "a", ran: &a}}, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunAll() = %v, want context.Canceled", err)
	}
	if a != 0 {
		t.Fatalf("step ran under cancelled ctx")
	}
}

func TestRunAllPreRun(t *testing.T) {
	t.Parallel()
	var ran, pre int
	step := preFake{fake{name: "p", ran: &ran}, &pre}
	if err := runner.RunAll(context.Background(), discardLog(), []program.Runner{step}, false); err != nil {
		t.Fatal(err)
	}
	if pre != 1 || ran != 1 {
		t.Fatalf("pre=%d ran=%d, want 1 1", pre, ran)
	}
}

func TestRunAllContinue(t *testing.T) {
	t.Parallel()
	var a, b, c int
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
	if a != 1 || b != 1 || c != 1 {
		t.Fatalf("ran a=%d b=%d c=%d, want 1 1 1", a, b, c)
	}
}

func TestRunAllContinueTwoFailures(t *testing.T) {
	t.Parallel()
	var a, b, c int
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
	if a != 1 || b != 1 || c != 1 {
		t.Fatalf("ran a=%d b=%d c=%d, want 1 1 1", a, b, c)
	}
}

func TestRunAllContinueAllOK(t *testing.T) {
	t.Parallel()
	var a int
	if err := runner.RunAll(context.Background(), discardLog(), []program.Runner{fake{name: "a", ran: &a}}, true); err != nil {
		t.Fatal(err)
	}
	if a != 1 {
		t.Fatalf("ran a=%d, want 1", a)
	}
}

func TestRunAllContinueCancelAfterFirst(t *testing.T) {
	t.Parallel()
	var a, b int
	ctx, cancel := context.WithCancel(context.Background())
	steps := []program.Runner{
		fake{name: "a", ran: &a},
		cancelFake{fake{name: "b", ran: &b}, cancel},
		fake{name: "c", ran: new(int)},
	}
	err := runner.RunAll(ctx, discardLog(), steps, true)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunAll() = %v, want context.Canceled", err)
	}
	if a != 1 || b != 1 {
		t.Fatalf("ran a=%d b=%d, want 1 1", a, b)
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
