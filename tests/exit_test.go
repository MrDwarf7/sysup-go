package tests

import (
	"errors"
	"testing"

	"sysup-go/cmd"
	"sysup-go/internal/mise"
	"sysup-go/internal/program"
	"sysup-go/internal/runner"
)

func TestExitCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: 0},
		{name: "unknown skip", err: &program.SkipError{Token: "nope"}, want: 3},
		{name: "unknown flag", err: errors.New("unknown flag: --bogus"), want: 2},
		{name: "other", err: errors.New("config: read missing"), want: 1},
		{name: "step failure", err: &runner.StepError{Name: "aur", Err: errors.New("exit 1")}, want: 5},
		{name: "mise up", err: &mise.Error{Op: "up", Err: errors.New("exit 1")}, want: 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cmd.ExitCode(tc.err)
			if got != tc.want {
				t.Fatalf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}
