package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

func filesystem(fsys afero.Fs) afero.Fs {
	if fsys == nil {
		return afero.NewOsFs()
	}
	return fsys
}

// AppDir returns the application config directory, creating it if needed.
// Path is os.UserConfigDir joined with AppName.
func AppDir(fsys afero.Fs) (string, error) {
	fsys = filesystem(fsys)
	base, err := os.UserConfigDir()
	if err != nil {
		return "", &Error{Op: "dir", Err: err}
	}
	dir := filepath.Join(base, AppName)
	if err := fsys.MkdirAll(dir, 0o755); err != nil {
		return "", &Error{Op: "dir", Path: dir, Err: err}
	}
	return dir, nil
}

// AppConfig returns AppDir joined with ConfigFileName.
func AppConfig(fsys afero.Fs) (string, error) {
	dir, err := AppDir(fsys)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFileName), nil
}
