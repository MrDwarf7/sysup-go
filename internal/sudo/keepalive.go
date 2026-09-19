// Package sudo keeps a sudo timestamp alive across a long plan.
package sudo

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"time"
)

const defaultInterval = 50 * time.Second

// Runner runs one external command. Tests inject fakes; production
// uses exec.CommandContext with stdin/out/err on the real process
// streams so `sudo -v` can prompt on the TTY.
type Runner func(ctx context.Context, name string, args ...string) error

// KeepAlive primes sudo then refreshes the timestamp on an interval
// until stop or ctx cancel.
type KeepAlive struct {
	Interval time.Duration
	Run      Runner
	Log      *slog.Logger
}

// Start runs `sudo -v` in the foreground (TTY password if needed),
// then ticks `sudo -n true` in a goroutine. stop cancels the ticker
// without cancelling the caller's ctx.
func (k KeepAlive) Start(ctx context.Context) (stop func(), err error) {
	run := k.Run
	if run == nil {
		run = defaultRun
	}
	log := k.Log
	if log == nil {
		log = slog.Default()
	}
	interval := k.Interval
	if interval <= 0 {
		interval = defaultInterval
	}

	if err := run(ctx, "sudo", "-v"); err != nil {
		return func() {}, &Error{Op: "prime", Err: err}
	}

	bg, cancel := context.WithCancel(ctx)
	go refresh(bg, run, log, interval)
	return cancel, nil
}

func refresh(ctx context.Context, run Runner, log *slog.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := run(ctx, "sudo", "-n", "true"); err != nil {
				log.WarnContext(ctx, "sudo keepalive refresh failed", "err", err)
			}
		}
	}
}

func defaultRun(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	// Real TTY so sudo -v can prompt and hide the password.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
