package tests

import (
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"
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

func TestConfigAppDirFromXDG(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	fsys := afero.NewOsFs()
	want := expectedAppDir(t, xdg)

	got, err := config.AppDir(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("AppDir = %q, want %q", got, want)
	}
	ok, err := afero.DirExists(fsys, want)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Errorf("AppDir did not create %q", want)
	}
}

func TestConfigAppDirNilFs(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	want := expectedAppDir(t, xdg)

	got, err := config.AppDir(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("AppDir = %q, want %q", got, want)
	}
}

func TestConfigAppDirCreatesMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg-missing")
	fsys := afero.NewMemMapFs()
	want := expectedAppDir(t, "/xdg-missing")

	got, err := config.AppDir(fsys)
	if err != nil {
		t.Fatalf("AppDir error = %v", err)
	}
	if got != want {
		t.Errorf("AppDir = %q, want %q", got, want)
	}
	ok, err := afero.DirExists(fsys, want)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Errorf("AppDir did not create %q", want)
	}
}

func TestConfigAppDirExists(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	fsys := afero.NewMemMapFs()
	want := expectedAppDir(t, "/xdg")
	if err := fsys.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := config.AppDir(fsys)
	if err != nil {
		t.Fatalf("AppDir error = %v", err)
	}
	if got != want {
		t.Errorf("AppDir = %q, want %q", got, want)
	}
}

func TestConfigAppConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	fsys := afero.NewMemMapFs()
	want := filepath.Join(expectedAppDir(t, "/xdg"), config.ConfigFileName)

	got, err := config.AppConfig(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("AppConfig = %q, want %q", got, want)
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

func TestConfigGenerateRoundTrip(t *testing.T) {
	t.Parallel()
	fsys := afero.NewMemMapFs()
	path := "/xdg/sysup/" + config.ConfigFileName
	if err := config.Generate(fsys, path); err != nil {
		t.Fatal(err)
	}
	got, err := config.Load(config.NewViper(fsys, path))
	if err != nil {
		t.Fatal(err)
	}
	want := config.Defaults()
	want.Cache.IncludeDirs = []string{}
	want.Cache.ExcludeDirs = []string{}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load(Generate) = %+v, want %+v", got, want)
	}
}

func TestConfigGenerateEmptyFile(t *testing.T) {
	t.Parallel()
	fsys := afero.NewMemMapFs()
	path := "/empty/" + config.ConfigFileName
	if err := afero.WriteFile(fsys, path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.Generate(fsys, path); err != nil {
		t.Fatal(err)
	}
	info, err := fsys.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("Generate left empty file")
	}
}

func TestConfigGenerateExists(t *testing.T) {
	t.Parallel()
	fsys := afero.NewMemMapFs()
	path := "/exists/" + config.ConfigFileName
	if err := afero.WriteFile(fsys, path, []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := config.Generate(fsys, path)
	requireConfigOp(t, err, "generate")
	if !errors.Is(err, fs.ErrExist) {
		t.Errorf("Unwrap = %v, want fs.ErrExist", err)
	}
	got, err := afero.ReadFile(fsys, path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "keep\n" {
		t.Errorf("clobbered file: %q", got)
	}
}

func TestConfigMissingOrEmpty(t *testing.T) {
	t.Parallel()
	fsys := afero.NewMemMapFs()
	path := "/x/" + config.ConfigFileName

	ok, err := config.MissingOrEmpty(fsys, path)
	if err != nil || !ok {
		t.Fatalf("missing: ok=%v err=%v, want true nil", ok, err)
	}

	if writeErr := afero.WriteFile(fsys, path, nil, 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	ok, err = config.MissingOrEmpty(fsys, path)
	if err != nil || !ok {
		t.Fatalf("empty: ok=%v err=%v, want true nil", ok, err)
	}

	if writeErr := afero.WriteFile(fsys, path, []byte("x"), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	ok, err = config.MissingOrEmpty(fsys, path)
	if err != nil || ok {
		t.Fatalf("non-empty: ok=%v err=%v, want false nil", ok, err)
	}

	dir := "/x/dir"
	if mkdirErr := fsys.MkdirAll(dir, 0o755); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	_, err = config.MissingOrEmpty(fsys, dir)
	requireConfigOp(t, err, "stat")
}

func TestConfigEncodeDurations(t *testing.T) {
	t.Parallel()
	b, err := config.Encode(config.Defaults())
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "50s") {
		t.Errorf("missing 50s interval:\n%s", s)
	}
	if !strings.Contains(s, "1m") {
		t.Errorf("missing 1m wait:\n%s", s)
	}
	if strings.Contains(s, "1m0s") {
		t.Errorf("uncompact wait:\n%s", s)
	}
}
