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

	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"sysup-go/internal/config"
	"sysup-go/internal/program"
)

// expectedAppDir is where AppDir looks. Unix honours XDG_CONFIG_HOME
// (unixXDG). Windows UserConfigDir is %AppData%; XDG is ignored.
func expectedAppDir(t *testing.T, unixXDG string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		base, err := os.UserConfigDir()
		if err != nil {
			t.Fatal(err)
		}
		return filepath.Join(base, config.AppName)
	}
	return filepath.Join(unixXDG, config.AppName)
}

func fixturePath(t *testing.T, parts ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	base := filepath.Join(filepath.Dir(file), "testdata")
	return filepath.Join(append([]string{base}, parts...)...)
}

// fixtureMapFS copies testdata into a MemMapFs rooted at the fixture
// directory. Paths stay relative (programs.toml, programs/*.toml) so
// program.Load sees the same layout as a BasePathFs over the app dir.
func fixtureMapFS(t *testing.T, parts ...string) afero.Fs {
	t.Helper()
	root := fixturePath(t, parts...)
	out := afero.NewMemMapFs()
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
			return out.MkdirAll(rel, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return afero.WriteFile(out, rel, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// rootedOsFs is the afero equivalent of os.DirFS(dir): every Open/Stat/Glob
// is relative to dir. Same wrapper loadSpecs uses around the app dir.
func rootedOsFs(dir string) afero.Fs {
	return afero.NewBasePathFs(afero.NewOsFs(), dir)
}

func loadProgramTree(t *testing.T, tree, backend string) ([]program.Spec, error) {
	t.Helper()
	root := fixturePath(t, "programs", tree)
	switch backend {
	case "mem":
		return program.Load(fixtureMapFS(t, "programs", tree))
	case "disk":
		return program.Load(rootedOsFs(root))
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
		return config.NewViper(afero.NewOsFs(), src), src
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
	return config.NewViper(fsys, "")
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
	env   map[string]string
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
	if !ok {
		return "", false
	}
	return v, true
}

func (f *fakeEnvPath) called() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}
