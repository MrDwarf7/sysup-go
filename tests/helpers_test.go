package tests

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"sysup-go/internal/config"
	"sysup-go/internal/program"
)

func fixturePath(t *testing.T, parts ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	base := filepath.Join(filepath.Dir(file), "testdata")
	return filepath.Join(append([]string{base}, parts...)...)
}

func fixtureMapFS(t *testing.T, parts ...string) fstest.MapFS {
	t.Helper()
	root := fixturePath(t, parts...)
	out := fstest.MapFS{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			out[rel] = &fstest.MapFile{Mode: fs.ModeDir}
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = &fstest.MapFile{Data: b}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func loadProgramTree(t *testing.T, tree, backend string) ([]program.Spec, error) {
	t.Helper()
	switch backend {
	case "mem":
		return program.Load(fixtureMapFS(t, "programs", tree))
	case "disk":
		return program.Load(os.DirFS(fixturePath(t, "programs", tree)))
	default:
		t.Fatalf("unknown backend %q", backend)
		return nil, nil
	}
}

func ioBackends() []string {
	return []string{"mem", "disk"}
}

func configViper(t *testing.T, fixture, backend string) (*viper.Viper, string) {
	t.Helper()
	src := fixturePath(t, "config", fixture)
	switch backend {
	case "mem":
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		path := "/" + config.ConfigFileName
		return memViper(t, map[string]string{path: string(b)}), path
	case "disk":
		v := viper.New()
		v.SetConfigFile(src)
		v.SetConfigType(config.ConfigType)
		return v, src
	default:
		t.Fatalf("unknown backend %q", backend)
		return nil, ""
	}
}

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

func requireConfigOp(t *testing.T, err error, op string) {
	t.Helper()
	if err == nil {
		t.Fatal("error = nil")
	}
	cfgErr, ok := errors.AsType[*config.Error](err)
	if !ok {
		t.Fatalf("error type %T (%v), want *config.Error", err, err)
	}
	if cfgErr.Op != op {
		t.Errorf("Op = %q, want %q", cfgErr.Op, op)
	}
}

func specNames(specs []program.Spec) []string {
	names := make([]string, len(specs))
	for i, s := range specs {
		names[i] = s.Name
	}
	return names
}

func discardLog() *slog.Logger {
	return slog.New(slog.DiscardHandler)
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
