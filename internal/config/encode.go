package config

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/afero"
)

// fileConfig is the on-disk TOML shape. Durations are strings so the
// file matches config.toml (`"50s"`, `"1m"`) instead of nanoseconds.
type fileConfig struct {
	Sudo       fileSudo       `toml:"sudo"`
	Mise       fileMise       `toml:"mise"`
	Cache      fileCache      `toml:"cache"`
	PkgManager filePkgManager `toml:"pkg_manager"`
	Shutdown   fileShutdown   `toml:"shutdown"`
	Retries    fileRetries    `toml:"retries"`
}

type fileSudo struct {
	Keepalive bool   `toml:"keepalive"`
	Interval  string `toml:"interval"`
}

type fileMise struct {
	Wrap              bool `toml:"wrap"`
	StripBinsFromPath bool `toml:"strip_bins_from_path"`
}

type fileCache struct {
	Enabled     bool     `toml:"enabled"`
	IncludeDirs []string `toml:"include_dirs"`
	ExcludeDirs []string `toml:"exclude_dirs"`
}

type filePkgManager struct {
	Name string `toml:"name"`
}

type fileShutdown struct {
	Force bool   `toml:"force"`
	Wait  string `toml:"wait"`
}

type fileRetries struct {
	Always      bool `toml:"always"`
	MaxAttempts int  `toml:"max_attempts"`
}

func toFile(cfg Config) fileConfig {
	inc := cfg.Cache.IncludeDirs
	if inc == nil {
		inc = []string{}
	}
	exc := cfg.Cache.ExcludeDirs
	if exc == nil {
		exc = []string{}
	}
	return fileConfig{
		Sudo: fileSudo{
			Keepalive: cfg.Sudo.Keepalive,
			Interval:  compactDuration(cfg.Sudo.Interval),
		},
		Mise: fileMise{
			Wrap:              cfg.Mise.Wrap,
			StripBinsFromPath: cfg.Mise.StripBinsFromPath,
		},
		Cache: fileCache{
			Enabled:     cfg.Cache.Enabled,
			IncludeDirs: inc,
			ExcludeDirs: exc,
		},
		PkgManager: filePkgManager{Name: cfg.PkgManager.Name},
		Shutdown: fileShutdown{
			Force: cfg.Shutdown.Force,
			Wait:  compactDuration(cfg.Shutdown.Wait),
		},
		Retries: fileRetries{
			Always:      cfg.Retries.Always,
			MaxAttempts: cfg.Retries.MaxAttempts,
		},
	}
}

func compactDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", d/time.Hour)
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", d/time.Minute)
	}
	if d%time.Second == 0 {
		return fmt.Sprintf("%ds", d/time.Second)
	}
	return d.String()
}

const fileHeader = "# sysup default config.\n"

// Encode marshals cfg as TOML with a short header comment.
func Encode(cfg Config) ([]byte, error) {
	body, err := toml.Marshal(toFile(cfg))
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(fileHeader)+len(body))
	out = append(out, fileHeader...)
	out = append(out, body...)
	return out, nil
}

// MissingOrEmpty is true when path does not exist or is a zero-byte file.
func MissingOrEmpty(fsys afero.Fs, path string) (bool, error) {
	fsys = filesystem(fsys)
	if path == "" {
		return false, &Error{Op: "stat", Err: errEmptyPath}
	}
	info, err := fsys.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return true, nil
		}
		return false, &Error{Op: "stat", Path: path, Err: err}
	}
	if info.IsDir() {
		return false, &Error{Op: "stat", Path: path, Err: errNotDir}
	}
	return info.Size() == 0, nil
}

// Write encodes cfg and writes it to path, creating parent dirs if needed.
func Write(fsys afero.Fs, path string, cfg Config) error {
	fsys = filesystem(fsys)
	if path == "" {
		return &Error{Op: "write", Err: errEmptyPath}
	}
	dir := filepath.Dir(path) // TEST: [platform] : pretty sure this fails on Windows in edge-cases
	if dir != "" && dir != "." {
		if err := fsys.MkdirAll(dir, 0o755); err != nil {
			return &Error{Op: "write", Path: path, Err: err}
		}
	}
	b, err := Encode(cfg)
	if err != nil {
		return &Error{Op: "encode", Path: path, Err: err}
	}
	if err := afero.WriteFile(fsys, path, b, 0o644); err != nil {
		return &Error{Op: "write", Path: path, Err: err}
	}
	return nil
}

// Generate writes Defaults to path when the file is missing or empty.
// A non-empty file is left untouched; the error unwraps to fs.ErrExist.
func Generate(fsys afero.Fs, path string) error {
	empty, err := MissingOrEmpty(fsys, path)
	if err != nil {
		return err
	}
	if !empty {
		return &Error{Op: "generate", Path: path, Err: fs.ErrExist}
	}
	return Write(fsys, path, Defaults())
}
