package resolve

// Error is a failure while resolving a package-manager binary.
type Error struct {
	Op  string
	Err error
}

func (e *Error) Error() string {
	if e.Op == "" {
		if e.Err != nil {
			return e.Err.Error()
		}
		return "resolve error"
	}
	if e.Err == nil {
		return e.Op
	}
	return e.Op + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

func wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Op: op, Err: err}
}
