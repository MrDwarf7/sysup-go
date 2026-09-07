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

func TestConfigLoad(t *testing.T) {
	t.Parallel()

	type tc struct {
		name    string
		fixture string
		want    config.Config
		wantErr string
	}
	cases := []tc{
		{
			name:    "valid full toml",
			fixture: "full.toml",
			want: func() config.Config {
				c := config.Defaults()
				c.Cache.IncludeDirs = []string{}
				c.Cache.ExcludeDirs = []string{}
				return c
			}(),
		},
		{
			name:    "partial overlay",
			fixture: "partial.toml",
			want: func() config.Config {
				c := config.Defaults()
				c.Sudo.Keepalive = false
				c.Mise.Wrap = false
				c.Cache.Enabled = false
				return c
			}(),
		},
		{
			name:    "bad duration",
			fixture: "bad-duration.toml",
			wantErr: "decode",
		},
		{
			name:    "unknown key",
			fixture: "unknown-key.toml",
			wantErr: "decode",
		},
	}

	for _, backend := range ioBackends() {
		for _, tt := range cases {
			t.Run(backend+"/"+tt.name, func(t *testing.T) {
				t.Parallel()
				v, path := configViper(t, tt.fixture, backend)
				v.SetConfigFile(path)
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

	t.Run("mem/no file", func(t *testing.T) {
		t.Parallel()
		got, err := config.Load(memViper(t, nil))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, config.Defaults()) {
			t.Errorf("Load() = %+v, want defaults", got)
		}
	})

	t.Run("mem/missing explicit path", func(t *testing.T) {
		t.Parallel()
		v := memViper(t, nil)
		v.SetConfigFile("/no/such/" + config.ConfigFileName)
		_, err := config.Load(v)
		requireConfigOp(t, err, "read")
	})
}

func TestConfigAppDirUsesFileDir(t *testing.T) {
	t.Parallel()
	path := fixturePath(t, "config", "mise-wrap.toml")
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	got, err := config.AppDir(v, afero.NewOsFs())
	if err != nil {
		t.Fatal(err)
	}
	want := fixturePath(t, "config")
	if got != want {
		t.Errorf("AppDir = %q, want %q", got, want)
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

	path := fixturePath(t, "config", "mise-wrap.toml")
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
