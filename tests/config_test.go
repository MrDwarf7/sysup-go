package tests

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"sysup-go/internal/config"
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

func TestConfigLoad(t *testing.T) {
	t.Parallel()
	file := "/" + config.ConfigFileName

	tests := []struct {
		name    string
		files   map[string]string
		path    string
		want    config.Config
		wantErr string
	}{
		{
			name: "no file",
			want: config.Defaults(),
		},
		{
			name:  "valid full toml",
			files: map[string]string{file: specTOML},
			path:  file,
			want: func() config.Config {
				c := config.Defaults()
				c.Cache.IncludeDirs = []string{}
				c.Cache.ExcludeDirs = []string{}
				return c
			}(),
		},
		{
			name: "partial overlay",
			files: map[string]string{file: `[sudo]
keepalive = false

[mise]
wrap = false

[cache]
enabled = false
`},
			path: file,
			want: func() config.Config {
				c := config.Defaults()
				c.Sudo.Keepalive = false
				c.Mise.Wrap = false
				c.Cache.Enabled = false
				return c
			}(),
		},
		{
			name: "bad duration",
			files: map[string]string{file: `[sudo]
interval = "nope"
`},
			path:    file,
			wantErr: "decode",
		},
		{
			name:    "missing explicit path",
			path:    "/no/such/" + config.ConfigFileName,
			wantErr: "read",
		},
		{
			name:    "unknown key",
			files:   map[string]string{file: "[foo]\nbar = 1\n"},
			path:    file,
			wantErr: "decode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := memViper(t, tt.files)
			if tt.path != "" {
				v.SetConfigFile(tt.path)
			}
			got, err := config.Load(v)
			if tt.wantErr != "" {
				requireConfigOp(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestConfigAppDirUsesFileDir(t *testing.T) {
	t.Parallel()
	fsys := afero.NewMemMapFs()
	dir := "/app"
	path := dir + "/" + config.ConfigFileName
	if err := fsys.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := afero.WriteFile(fsys, path, []byte("[mise]\nwrap = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := viper.New()
	v.SetFs(fsys)
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	got, err := config.AppDir(v, fsys)
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Errorf("AppDir = %q, want %q", got, dir)
	}
}

func TestConfigAppDirMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg-missing")
	_, err := config.AppDir(nil, afero.NewMemMapFs())
	if err == nil {
		t.Fatal("expected error")
	}
	var ce *config.Error
	if !errors.As(err, &ce) {
		t.Fatalf("got %T %v, want *config.Error", err, err)
	}
	if ce.Op != "dir" {
		t.Errorf("Op = %q, want dir", ce.Op)
	}
}

func TestConfigAppDirExists(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	fsys := afero.NewMemMapFs()
	want := "/xdg/" + config.AppName
	if err := fsys.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := config.AppDir(nil, fsys)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("AppDir = %q, want %q", got, want)
	}
}

func TestConfigOriginFrom(t *testing.T) {
	t.Parallel()
	if o := config.OriginFrom(nil); o.Kind != config.KindDefault {
		t.Errorf("nil viper Kind = %v, want KindDefault", o.Kind)
	}
	v := viper.New()
	if o := config.OriginFrom(v); o.Kind != config.KindDefault || o.Path != "" {
		t.Errorf("empty viper = %+v, want default", o)
	}

	fsys := afero.NewMemMapFs()
	path := "/app/" + config.ConfigFileName
	if err := fsys.MkdirAll("/app", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := afero.WriteFile(fsys, path, []byte("[mise]\nwrap = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v.SetFs(fsys)
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	o := config.OriginFrom(v)
	if o.Kind != config.KindFile || o.Path != path {
		t.Errorf("Origin = %+v, want file %q", o, path)
	}
}

func TestConfigDefaults(t *testing.T) {
	t.Parallel()
	c := config.Defaults()
	if !c.Sudo.Keepalive || c.Sudo.Interval != 50*time.Second {
		t.Errorf("sudo = %+v", c.Sudo)
	}
	if !c.Mise.Wrap || !c.Cache.Enabled || c.Shutdown.Force || c.Shutdown.Wait != time.Minute {
		t.Errorf("defaults = %+v", c)
	}
}
