package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

func locate() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", &Error{Op: "dir", Err: err}
	}
	return filepath.Join(base, AppName), nil
}

func AppDir(v *viper.Viper, fsys afero.Fs) (string, error) {
	if fsys == nil {
		fsys = afero.NewOsFs()
	}

	dir, err := dirFrom(v)
	if err != nil {
		return "", err
	}

	info, err := fsys.Stat(dir)
	if err != nil {
		return "", &Error{Op: "dir", Path: dir, Err: err}
	}
	if !info.IsDir() {
		return "", &Error{Op: "dir", Path: dir, Err: errNotDir}
	}
	return dir, nil
}

func dirFrom(v *viper.Viper) (string, error) {
	if v == nil {
		return locate()
	}
	used := v.ConfigFileUsed()
	if used == "" {
		return locate()
	}
	return filepath.Dir(used), nil
}
