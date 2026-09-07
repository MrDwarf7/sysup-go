package config

import "fmt"

// Error is a config directory or file failure.
type Error struct {
	Op   string // "dir", "read", "decode"
	Path string
	Err  error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("config: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("config: %s %s: %v", e.Op, e.Path, e.Err)
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Err
}
