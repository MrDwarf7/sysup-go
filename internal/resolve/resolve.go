// Package resolve selects a package-manager binary from env, PATH, and config.
package resolve

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// LookPath finds an executable on PATH. It matches exec.LookPath.
type LookPath func(file string) (string, error)

// Env looks up an environment variable. It matches os.LookupEnv.
type Env func(key string) (string, bool)

// Resolver locates a package-manager binary using injected lookups.
type Resolver struct {
	LookPath  LookPath
	LookupEnv Env
}

const envPkgManager = "PKG_MANAGER"

// PkgManager returns the path of the package-manager binary.
func (r Resolver) PkgManager(ctx context.Context, cfgName string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", wrap("pkg_manager", err)
	}

	if r.LookupEnv != nil {
		val, ok := r.LookupEnv(envPkgManager)
		if ok {
			if val == "" {
				return "", wrap("env", errors.New("PKG_MANAGER is empty"))
			}
			path, err := r.LookPath(val)
			if err != nil {
				return "", wrap("lookpath", fmt.Errorf("%s: %w", val, err))
			}
			if err := ctx.Err(); err != nil {
				return "", wrap("pkg_manager", err)
			}
			return path, nil
		}
	}

	paru, yay, err := r.probeParuYay(ctx)
	if err != nil {
		return "", wrap("pkg_manager", err)
	}
	if paru != "" {
		return paru, nil
	}
	if yay != "" {
		return yay, nil
	}

	if cfgName != "" {
		path, err := r.LookPath(cfgName)
		if err != nil {
			return "", wrap("lookpath", fmt.Errorf("%s: %w", cfgName, err))
		}
		if err := ctx.Err(); err != nil {
			return "", wrap("pkg_manager", err)
		}
		return path, nil
	}

	return "", wrap("pkg_manager", errors.New("cannot resolve a package manager"))
}

func (r Resolver) probeParuYay(ctx context.Context) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}

	type result struct {
		paru string
		yay  string
	}
	ch := make(chan result, 1)
	go func() {
		var (
			wg  sync.WaitGroup
			mu  sync.Mutex
			out result
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			path, err := r.LookPath("paru")
			if err != nil {
				return
			}
			mu.Lock()
			out.paru = path
			mu.Unlock()
		}()
		go func() {
			defer wg.Done()
			path, err := r.LookPath("yay")
			if err != nil {
				return
			}
			mu.Lock()
			out.yay = path
			mu.Unlock()
		}()
		wg.Wait()
		ch <- out
	}()

	select {
	case <-ctx.Done():
		return "", "", ctx.Err()
	case out := <-ch:
		if err := ctx.Err(); err != nil {
			return "", "", err
		}
		return out.paru, out.yay, nil
	}
}
