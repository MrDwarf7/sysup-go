package cmd

import (
	"errors"
	"testing"

	"sysup-go/internal/program"
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
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := exitCode(tc.err)
			if got != tc.want {
				t.Fatalf("exitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}
