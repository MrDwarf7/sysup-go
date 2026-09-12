package tests

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"sysup-go/internal/logfmt"
)

func TestLogfmtOrder(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := logfmt.New(&buf, logfmt.Options{Level: slog.LevelDebug})
	rec := slog.NewRecord(time.Date(2026, 9, 12, 10, 40, 7, 430000000, time.FixedZone("AEST", 10*3600)), slog.LevelInfo, "running", 0)
	rec.Add("step", "pacman")
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if strings.Contains(got, "\x1b") {
		t.Fatalf("plain handler wrote ANSI: %q", got)
	}
	wantPrefix := "INFO  | 2026-09-12T10:40:07.430+10:00 | running"
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("got %q, want prefix %q", got, wantPrefix)
	}
	if !strings.Contains(got, "step=pacman") {
		t.Fatalf("missing attr: %q", got)
	}
	info := strings.Index(got, "INFO")
	pipe1 := strings.Index(got, "|")
	msg := strings.Index(got, "running")
	if info >= pipe1 || pipe1 >= msg {
		t.Fatalf("want level | time | msg, got %q", got)
	}
}

func TestLogfmtColor(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := logfmt.New(&buf, logfmt.Options{Level: slog.LevelInfo, Color: true})
	rec := slog.NewRecord(time.Now(), slog.LevelWarn, "hot", 0)
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "\x1b[") {
		t.Fatalf("color handler wrote no ANSI: %q", buf.String())
	}
}

func TestLogfmtDefaultFile(t *testing.T) {
	t.Parallel()
	p := logfmt.DefaultFile()
	if p == "" || !strings.Contains(p, "sysup.log") {
		t.Fatalf("DefaultFile = %q", p)
	}
}

func TestLogfmtDisabled(t *testing.T) {
	t.Parallel()
	if !logfmt.Disabled("") || !logfmt.Disabled("-") {
		t.Fatal("empty and - should disable")
	}
	if logfmt.Disabled("/tmp/sysup.log") {
		t.Fatal("path should stay enabled")
	}
}
