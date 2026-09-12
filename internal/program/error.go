package program

import "fmt"

type Error struct {
	Op   string
	Path string
	Err  error
}

func (e *Error) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("program: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("program: %s %s: %v", e.Op, e.Path, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

type SkipError struct {
	Token string
}

func (e *SkipError) Error() string {
	return fmt.Sprintf("unknown skip token %q", e.Token)
}
