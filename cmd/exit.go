package cmd

import (
	"errors"
	"strings"

	"sysup-go/internal/program"
)

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var skipErr *program.SkipError
	if errors.As(err, &skipErr) {
		return 3
	}
	msg := err.Error()
	if strings.Contains(msg, "unknown flag") || strings.Contains(msg, "unknown shorthand flag") {
		return 2
	}
	return 1
}
