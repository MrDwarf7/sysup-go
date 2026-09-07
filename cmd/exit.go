package cmd

import (
	"errors"
	"strings"

	"sysup-go/internal/program"
	"sysup-go/internal/runner"
)

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var skipErr *program.SkipError
	if errors.As(err, &skipErr) {
		return 3
	}
	var stepErr *runner.StepError
	if errors.As(err, &stepErr) {
		return 5
	}
	msg := err.Error()
	if strings.Contains(msg, "unknown flag") || strings.Contains(msg, "unknown shorthand flag") {
		return 2
	}
	return 1
}
