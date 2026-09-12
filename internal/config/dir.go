package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// Dir returns the application config directory path. It does not create
// the directory. Path is os.UserConfigDir joined with AppName.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", &Error{Op: "dir", Err: err}
	}
	return filepath.Join(base, AppName), nil
}

// AppDir returns Dir, creating it if needed.
func AppDir(fsys afero.Fs) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := fsys.MkdirAll(dir, 0o755); err != nil {
		return "", &Error{Op: "dir", Path: dir, Err: err}
	}
	return dir, nil
}

// AppConfig returns Dir joined with ConfigFileName. It does not create
// the directory; Generate / Write create parents when they write.
func AppConfig() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFileName), nil
}
