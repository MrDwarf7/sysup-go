package program

import "context"

type Runner interface {
	Run(ctx context.Context) error
	Meta() Spec
}

type PreRunner interface {
	PreRun(ctx context.Context) error
}

type PostRunner interface {
	PostRun(ctx context.Context) error
}

type Program interface {
	Runner
	PreRunner
	PostRunner
}
