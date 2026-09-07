package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

const specTOML = `[sudo]
keepalive = true
interval = "50s"

[mise]
wrap = true

[cache]
enabled = true
include_dirs = []
exclude_dirs = []

[pkg_manager]
name = ""

[shutdown]
force = false
wait = "1m"
`

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		want    Config
		wantErr string
		wantNil bool
	}{
		{
			name: "no file in temp XDG",
			setup: func(t *testing.T) string {
				xdg := t.TempDir()
				t.Setenv("XDG_CONFIG_HOME", xdg)
				return ""
			},
			want:    Defaults(),
			wantNil: true,
		},
		{
			name: "valid full toml",
			setup: func(t *testing.T) string {
				writeXDGConfig(t, specTOML)
				return ""
			},
			want: func() Config {
				c := Defaults()
				c.Cache.IncludeDirs = []string{}
				c.Cache.ExcludeDirs = []string{}
				return c
			}(),
		},
		{
			name: "false flags apply",
			setup: func(t *testing.T) string {
				writeXDGConfig(t, `[sudo]
keepalive = false

[mise]
wrap = false

[cache]
enabled = false
`)
				return ""
			},
			want: func() Config {
				c := Defaults()
				c.Sudo.Keepalive = false
				c.Mise.Wrap = false
				c.Cache.Enabled = false
				return c
			}(),
		},
		{
			name: "bad duration",
			setup: func(t *testing.T) string {
				return writeXDGConfig(t, `[sudo]
interval = "nope"
`)
			},
			wantErr: "decode",
		},
		{
			name: "missing explicit path",
			setup: func(t *testing.T) string {
				t.Setenv("XDG_CONFIG_HOME", t.TempDir())
				return "/no/such/file.toml"
			},
			wantErr: "read",
		},
		{
			name: "unknown key",
			setup: func(t *testing.T) string {
				return writeXDGConfig(t, "[foo]\nbar = 1\n")
			},
			wantErr: "decode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			got, err := Load(path)
			if tt.wantErr != "" {
				cfgErr := requireError(t, err, tt.wantErr)
				if path != "" && cfgErr.Path != path {
					t.Errorf("Path = %q, want %q", cfgErr.Path, path)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load() = %+v, want %+v", got, tt.want)
			}
			if tt.wantNil {
				assertDefaultFields(t, got)
				dir, dirErr := Dir()
				if dirErr != nil {
					t.Fatalf("Dir() error = %v", dirErr)
				}
				if _, statErr := os.Stat(dir); !errors.Is(statErr, os.ErrNotExist) {
					t.Errorf("Load created %s", dir)
				}
			}
		})
	}
}

func TestDir(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error = %v", err)
	}
	want := filepath.Join(xdg, "sysup-go")
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

func writeXDGConfig(t *testing.T, content string) string {
	t.Helper()
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, "sysup-go")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func requireError(t *testing.T, err error, op string) *Error {
	t.Helper()
	if err == nil {
		t.Fatal("Load() error = nil")
	}
	var cfgErr *Error
	if !errors.As(err, &cfgErr) {
		t.Fatalf("Load() error type %T (%v), want *Error", err, err)
	}
	if cfgErr.Op != op {
		t.Errorf("Op = %q, want %q", cfgErr.Op, op)
	}
	if cfgErr.Path == "" {
		t.Error("Path is empty")
	}
	return cfgErr
}

func assertDefaultFields(t *testing.T, c Config) {
	t.Helper()
	if !c.Sudo.Keepalive {
		t.Error("Sudo.Keepalive = false, want true")
	}
	if c.Sudo.Interval != 50*time.Second {
		t.Errorf("Sudo.Interval = %v, want 50s", c.Sudo.Interval)
	}
	if !c.Mise.Wrap {
		t.Error("Mise.Wrap = false, want true")
	}
	if !c.Cache.Enabled {
		t.Error("Cache.Enabled = false, want true")
	}
	if c.Shutdown.Wait != time.Minute {
		t.Errorf("Shutdown.Wait = %v, want 1m", c.Shutdown.Wait)
	}
	if c.Shutdown.Force {
		t.Error("Shutdown.Force = true, want false")
	}
}
