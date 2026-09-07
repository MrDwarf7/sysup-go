package cmd

import (
	"errors"
	"strings"

	"sysup-go/internal/program"
	"sysup-go/internal/runner"
)

// Maps a command error to the process status. Cobra's Execute returns
// error; the OS still needs a number. A function so tests can drive the
// table without os.Exit killing the test process.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if _, ok := errors.AsType[*program.SkipError](err); ok {
		return 3
	}
	if _, ok := errors.AsType[*runner.StepError](err); ok {
		return 5
	}
	msg := err.Error()
	if strings.Contains(msg, "unknown flag") || strings.Contains(msg, "unknown shorthand flag") {
		return 2
	}
	return 1
}
