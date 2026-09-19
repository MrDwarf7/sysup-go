package sudo_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"sysup-go/internal/sudo"
)

func TestKeepAlivePrimeThenStop(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var calls [][]string
	run := func(_ context.Context, name string, args ...string) error {
		mu.Lock()
		calls = append(calls, append([]string{name}, args...))
		mu.Unlock()
		return nil
	}
	ka := sudo.KeepAlive{
		Interval: time.Hour,
		Run:      run,
		Log:      slog.Default(),
	}
	stop, err := ka.Start(t.Context())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	stop()
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 1 {
		t.Fatalf("calls = %v, want only sudo -v", calls)
	}
	if got := strings.Join(calls[0], " "); got != "sudo -v" {
		t.Fatalf("first call = %q, want sudo -v", got)
	}
}

func TestKeepAliveRefreshes(t *testing.T) {
	t.Parallel()
	var n atomic.Int32
	run := func(_ context.Context, _ string, _ ...string) error {
		n.Add(1)
		return nil
	}
	ka := sudo.KeepAlive{
		Interval: 20 * time.Millisecond,
		Run:      run,
		Log:      slog.Default(),
	}
	stop, err := ka.Start(t.Context())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(70 * time.Millisecond)
	stop()
	if got := n.Load(); got < 3 {
		t.Fatalf("runs = %d, want >= 3 (prime + refreshes)", got)
	}
}

func TestKeepAlivePrimeFailure(t *testing.T) {
	t.Parallel()
	var n atomic.Int32
	boom := errors.New("no tty")
	run := func(_ context.Context, _ string, _ ...string) error {
		n.Add(1)
		return boom
	}
	ka := sudo.KeepAlive{Run: run, Log: slog.Default()}
	stop, err := ka.Start(t.Context())
	if err == nil {
		stop()
		t.Fatal("Start = nil, want prime error")
	}
	var se *sudo.Error
	if !errors.As(err, &se) {
		t.Fatalf("got %T %v, want *sudo.Error", err, err)
	}
	if se.Op != "prime" {
		t.Errorf("Op = %q, want prime", se.Op)
	}
	time.Sleep(30 * time.Millisecond)
	if got := n.Load(); got != 1 {
		t.Fatalf("runs after failed prime = %d, want 1", got)
	}
}

func TestKeepAliveStopHaltsRefreshes(t *testing.T) {
	t.Parallel()
	var n atomic.Int32
	run := func(_ context.Context, _ string, _ ...string) error {
		n.Add(1)
		return nil
	}
	ka := sudo.KeepAlive{
		Interval: 15 * time.Millisecond,
		Run:      run,
		Log:      slog.Default(),
	}
	stop, err := ka.Start(t.Context())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(40 * time.Millisecond)
	stop()
	after := n.Load()
	time.Sleep(50 * time.Millisecond)
	if got := n.Load(); got != after {
		t.Fatalf("runs grew after stop: %d -> %d", after, got)
	}
}
