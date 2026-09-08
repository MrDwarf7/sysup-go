package config

import (
	"fmt"
)

// errNotDir   = errors.New("not a directory")
// var errNilViper = errors.New("nil viper")

type Error struct {
	Op   string
	Path string
	Err  error
}

func (e *Error) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("config: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("config: %s %s: %v", e.Op, e.Path, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}
