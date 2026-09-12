// Package logfmt is a slog.Handler that prints level | time | msg.
// Level is tinted on a TTY. The file handler stays plain so logs
// stay grepable.
package logfmt

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const timeLayout = "2006-01-02T15:04:05.000Z07:00"

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiGray   = "\x1b[90m"
	ansiCyan   = "\x1b[36m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
)

// DefaultFile is os.TempDir()/sysup.log. os.TempDir covers Unix TMPDIR,
// Windows GetTempPath, and macOS (no $TEMP; TMPDIR or /tmp).
func DefaultFile() string {
	return filepath.Join(os.TempDir(), "sysup.log")
}

// Options configure Handler.
type Options struct {
	Level slog.Level
	Color bool
}

// Handler writes records as "LEVEL | time | msg key=val".
type Handler struct {
	w     io.Writer
	opts  Options
	attrs []slog.Attr
	group string
	mu    *sync.Mutex
}

// New returns a Handler writing to w.
func New(w io.Writer, opts Options) *Handler {
	return &Handler{w: w, opts: opts, mu: new(sync.Mutex)}
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	level := r.Level.String()
	if h.opts.Color {
		b.WriteString(levelColor(r.Level))
		fmt.Fprintf(&b, "%-5s", level)
		b.WriteString(ansiReset)
	} else {
		fmt.Fprintf(&b, "%-5s", level)
	}
	b.WriteString(" | ")
	if h.opts.Color {
		b.WriteString(ansiGray)
	}
	ts := r.Time
	if ts.IsZero() {
		ts = time.Now()
	}
	b.WriteString(ts.Format(timeLayout))
	if h.opts.Color {
		b.WriteString(ansiReset)
	}
	b.WriteString(" | ")
	b.WriteString(r.Message)
	for _, a := range h.attrs {
		writeAttr(&b, h.group, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		writeAttr(&b, h.group, a)
		return true
	})
	b.WriteByte('\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, b.String())
	return err
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cp := *h
	cp.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &cp
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	cp := *h
	if cp.group == "" {
		cp.group = name
	} else {
		cp.group = cp.group + "." + name
	}
	return &cp
}

func writeAttr(b *strings.Builder, group string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	key := a.Key
	if group != "" {
		key = group + "." + key
	}
	b.WriteByte(' ')
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(a.Value.String())
}

func levelColor(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return ansiBold + ansiRed
	case level >= slog.LevelWarn:
		return ansiBold + ansiYellow
	case level >= slog.LevelInfo:
		return ansiBold + ansiGreen
	default:
		return ansiCyan
	}
}

// ColorTTY is true when f is a character device (a real terminal).
func ColorTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// Disabled reports a --log-file value that means "no file".
func Disabled(path string) bool {
	return path == "" || path == "-"
}
