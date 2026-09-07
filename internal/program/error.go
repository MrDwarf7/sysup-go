package program

import "fmt"

// Error is a typed failure from Load.
type Error struct {
	Op   string // discover, read, decode, validate
	Path string
	Err  error
}

func (e *Error) Error() string {
	if e.Path != "" && e.Err != nil {
		return fmt.Sprintf("program: %s %s: %v", e.Op, e.Path, e.Err)
	}
	if e.Path != "" {
		return fmt.Sprintf("program: %s %s", e.Op, e.Path)
	}
	if e.Err != nil {
		return fmt.Sprintf("program: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("program: %s", e.Op)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// SkipError is returned by Filter when a skip token matches no spec.
type SkipError struct {
	Token string
}

func (e *SkipError) Error() string {
	return fmt.Sprintf("unknown skip token %q", e.Token)
}
