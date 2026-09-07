package runner

import "fmt"

// StepError is a failure from one plan step.
type StepError struct {
	Name string
	Err  error
}

func (e *StepError) Error() string {
	if e.Name == "" {
		return fmt.Sprintf("step: %v", e.Err)
	}
	return fmt.Sprintf("step %s: %v", e.Name, e.Err)
}

func (e *StepError) Unwrap() error {
	return e.Err
}
