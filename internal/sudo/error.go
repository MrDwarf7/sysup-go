package sudo

import "fmt"

// Error is a keepalive failure (usually prime `sudo -v`).
type Error struct {
	Op  string
	Err error
}

func (e *Error) Error() string {
	if e == nil {
		return "sudo: <nil>"
	}
	if e.Op == "" {
		return fmt.Sprintf("sudo: %v", e.Err)
	}
	return fmt.Sprintf("sudo: %s: %v", e.Op, e.Err)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
