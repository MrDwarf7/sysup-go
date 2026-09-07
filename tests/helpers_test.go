package tests

import (
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"sysup-go/internal/config"
	"sysup-go/internal/program"
)

func memViper(t *testing.T, files map[string]string) *viper.Viper {
	t.Helper()
	fsys := afero.NewMemMapFs()
	for path, body := range files {
		if err := afero.WriteFile(fsys, path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	v := viper.New()
	v.SetFs(fsys)
	v.SetConfigType(config.ConfigType)
	return v
}

func requireConfigOp(t *testing.T, err error, op string) *config.Error {
	t.Helper()
	if err == nil {
		t.Fatal("error = nil")
	}
	var cfgErr *config.Error
	if !errors.As(err, &cfgErr) {
		t.Fatalf("error type %T (%v), want *config.Error", err, err)
	}
	if cfgErr.Op != op {
		t.Errorf("Op = %q, want %q", cfgErr.Op, op)
	}
	return cfgErr
}

func specNames(specs []program.Spec) []string {
	names := make([]string, len(specs))
	for i, s := range specs {
		names[i] = s.Name
	}
	return names
}

func ptr(s string) *string { return &s }

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeEnvPath struct {
	env   map[string]*string
	paths map[string]string

	mu    sync.Mutex
	calls []string
}

func (f *fakeEnvPath) lookPath(file string) (string, error) {
	f.mu.Lock()
	f.calls = append(f.calls, file)
	f.mu.Unlock()
	if p, ok := f.paths[file]; ok {
		return p, nil
	}
	return "", errors.New("not found")
}

func (f *fakeEnvPath) lookupEnv(key string) (string, bool) {
	v, ok := f.env[key]
	if !ok || v == nil {
		return "", false
	}
	return *v, true
}

func (f *fakeEnvPath) called() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}
