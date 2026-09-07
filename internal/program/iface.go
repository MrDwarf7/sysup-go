package program

import "context"

// Runner runs one plan step.
type Runner interface {
	Run(ctx context.Context) error
}

// PreRunner is an optional hook before Run.
type PreRunner interface {
	PreRun(ctx context.Context) error
}

// PostRunner is an optional hook after Run.
type PostRunner interface {
	PostRun(ctx context.Context) error
}

// Program is a step that implements Run and both hooks.
type Program interface {
	Runner
	PreRunner
	PostRunner
}
