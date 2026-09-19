package runner

import (
	"fmt"
	"strings"
)

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

// ContinueError is the -c / continue summary: every failed step, in order.
type ContinueError struct {
	Steps []StepError
}

func (e *ContinueError) Error() string {
	var b strings.Builder
	for i := range e.Steps {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(e.Steps[i].Error())
	}
	return b.String()
}

func (e *ContinueError) Unwrap() []error {
	out := make([]error, len(e.Steps))
	for i := range e.Steps {
		out[i] = &e.Steps[i]
	}
	return out
}
