package mise

import "fmt"

type Error struct {
	Op  string
	Err error
}

func (e *Error) Error() string {
	if e.Op == "" {
		return fmt.Sprintf("mise: %v", e.Err)
	}
	return fmt.Sprintf("mise: %s: %v", e.Op, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}
