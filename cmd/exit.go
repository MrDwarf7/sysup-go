package cmd

import (
	"errors"

	"sysup-go/internal/mise"
	"sysup-go/internal/program"
	"sysup-go/internal/runner"
)

// FlagError is a cobra flag-parse failure. SetFlagErrorFunc wraps
// those so ExitCode does not string-match the message.
type FlagError struct {
	Err error
}

func (e *FlagError) Error() string {
	if e.Err == nil {
		return "flag error"
	}
	return e.Err.Error()
}

func (e *FlagError) Unwrap() error {
	return e.Err
}

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
	if _, ok := errors.AsType[*runner.ContinueError](err); ok {
		return 5
	}
	if _, ok := errors.AsType[*runner.StepError](err); ok {
		return 5
	}
	if _, ok := errors.AsType[*mise.Error](err); ok {
		return 5
	}
	if _, ok := errors.AsType[*FlagError](err); ok {
		return 2
	}
	return 1
}
