package resolve

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type LookPath func(file string) (string, error)

type Env func(key string) (string, bool)

type Resolver struct {
	LookPath  LookPath
	LookupEnv Env
}

func (r Resolver) PkgManager(ctx context.Context, cfgName string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", wrap("pkg_manager", err)
	}

	path, handled, err := r.fromEnv(ctx)
	if err != nil {
		return "", err
	}
	if handled {
		return path, nil
	}

	path, err = r.probeFallbacks(ctx)
	if err != nil {
		return "", wrap("pkg_manager", err)
	}
	if path != "" {
		return path, nil
	}

	if cfgName == "" {
		return "", wrap("pkg_manager", errors.New("cannot resolve a package manager"))
	}
	return r.look(ctx, cfgName)
}

func (r Resolver) fromEnv(ctx context.Context) (string, bool, error) {
	if r.LookupEnv == nil {
		return "", false, nil
	}
	val, ok := r.LookupEnv(EnvPkgManager)
	if !ok {
		return "", false, nil
	}
	if val == "" {
		return "", true, wrap("env", errors.New("PKG_MANAGER is empty"))
	}
	path, err := r.look(ctx, val)
	return path, true, err
}

func (r Resolver) look(ctx context.Context, name string) (string, error) {
	path, err := r.LookPath(name)
	if err != nil {
		return "", wrap("lookpath", fmt.Errorf("%s: %w", name, err))
	}
	if err := ctx.Err(); err != nil {
		return "", wrap("pkg_manager", err)
	}
	return path, nil
}

func (r Resolver) probeFallbacks(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	paths := make([]string, len(FallbackHelpers))
	var wg sync.WaitGroup
	for i, helper := range FallbackHelpers {
		wg.Add(1)
		go func(i int, helper Helper) {
			defer wg.Done()
			path, err := r.LookPath(helper.String())
			if err != nil {
				return
			}
			paths[i] = path
		}(i, helper)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-done:
	}

	for _, path := range paths {
		if path != "" {
			return path, nil
		}
	}
	return "", nil
}
