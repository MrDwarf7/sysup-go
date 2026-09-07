package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

const (
	appName        = "sysup-go"
	configFileName = "config.toml"
)

// Dir returns os.UserConfigDir joined with "sysup-go".
// It does not create the directory.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", &Error{Op: "dir", Err: err}
	}
	return filepath.Join(base, appName), nil
}

// Load decodes config.toml at path.
// An empty path means Dir()/config.toml; a missing default file
// returns Defaults. A missing explicit path is an error.
func Load(path string) (Config, error) {
	explicit := path != ""
	if !explicit {
		dir, err := Dir()
		if err != nil {
			return Config{}, err
		}
		path = filepath.Join(dir, configFileName)
	}

	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicit {
			return Defaults(), nil
		}
		return Config{}, &Error{Op: "read", Path: path, Err: err}
	}

	cfg := Defaults()
	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		op := "read"
		var parseErr viper.ConfigParseError
		if errors.As(err, &parseErr) {
			op = "decode"
		}
		return Config{}, &Error{Op: op, Path: path, Err: err}
	}

	err := v.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.ErrorUnused = true
		dc.DecodeHook = mapstructure.StringToTimeDurationHookFunc()
	})
	if err != nil {
		return Config{}, &Error{Op: "decode", Path: path, Err: err}
	}
	return cfg, nil
}
