package resolve

import (
	"context"
	"errors"
	"fmt"
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

	path, handled, err := r.fromEnv()
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
	return r.look(cfgName)
}

func (r Resolver) fromEnv() (string, bool, error) {
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
	path, err := r.look(val)
	return path, true, err
}

func (r Resolver) look(name string) (string, error) {
	path, err := r.LookPath(name)
	if err != nil {
		return "", wrap("lookpath", fmt.Errorf("%s: %w", name, err))
	}
	return path, nil
}

type hit struct {
	helper string
	path   string
}

func (r Resolver) probeFallbacks(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	found, err := collectHits(ctx, startProbes(ctx, r.LookPath))
	if err != nil {
		return "", err
	}
	return pickHelper(found), nil
}

func startProbes(ctx context.Context, look LookPath) <-chan hit {
	ch := make(chan hit, len(FallbackHelpers))
	for _, helper := range FallbackHelpers {
		go func(h string) {
			path, err := look(h)
			if err != nil {
				path = ""
			}
			select {
			case ch <- hit{helper: h, path: path}:
			case <-ctx.Done():
			}
		}(helper)
	}
	return ch
}

func collectHits(ctx context.Context, ch <-chan hit) (map[string]string, error) {
	found := make(map[string]string, len(FallbackHelpers))
	for range FallbackHelpers {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case h := <-ch:
			if h.path != "" {
				found[h.helper] = h.path
			}
		}
	}
	return found, nil
}

func pickHelper(found map[string]string) string {
	for _, helper := range FallbackHelpers {
		if path := found[helper]; path != "" {
			return path
		}
	}
	return ""
}
