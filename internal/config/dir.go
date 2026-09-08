package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// Takes an optional afero.Fs, creating one if nil.
// If either is nil, defaults are attempted.
// Returns the path to the application config directory, creating it if necessary.
func AppDir(fsys afero.Fs) (string, error) {
	if fsys == nil {
		fsys = afero.NewOsFs()
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", &Error{Op: "dir", Err: err}
	}
	dir := filepath.Join(base, AppName)
	if dir == "" {
		return "", &Error{Op: "dir", Path: dir, Err: os.ErrNotExist}
	}

	if exists, err := afero.DirExists(fsys, dir); err != nil {
		return "", &Error{Op: "dir", Path: dir, Err: err}
	} else if !exists {
		if err := fsys.MkdirAll(dir, os.FileMode(0o755)); err != nil {
			return "", &Error{Op: "dir", Path: dir, Err: err}
		}
	}
	return dir, nil
}

// Calls AppDir first so the call is idempotent with respect to
// directory creation. The returned path is what viper reports via
// ConfigFileUsed() after a successful ReadInConfig.
func AppConfig(fsys afero.Fs) (string, error) {
	dir, err := AppDir(fsys)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigName+"."+ConfigType), nil
}
