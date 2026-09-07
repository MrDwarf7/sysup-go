// Package config loads XDG config.toml for sysup-go.
package config

import "time"

// Config is the decoded contents of config.toml.
type Config struct {
	Sudo       Sudo       `mapstructure:"sudo"`
	Mise       Mise       `mapstructure:"mise"`
	Cache      Cache      `mapstructure:"cache"`
	PkgManager PkgManager `mapstructure:"pkg_manager"`
	Shutdown   Shutdown   `mapstructure:"shutdown"`
}

// Sudo is the sudo keepalive wrap.
type Sudo struct {
	Keepalive bool          `mapstructure:"keepalive"`
	Interval  time.Duration `mapstructure:"interval"` // default 50s
}

// Mise is the mise PATH wrap.
type Mise struct {
	Wrap bool `mapstructure:"wrap"` // default true
}

// Cache is the end-of-run directory sweep.
type Cache struct {
	Enabled     bool     `mapstructure:"enabled"` // default true
	IncludeDirs []string `mapstructure:"include_dirs"`
	ExcludeDirs []string `mapstructure:"exclude_dirs"`
}

// PkgManager is the last-resort package manager name.
type PkgManager struct {
	Name string `mapstructure:"name"`
}

// Shutdown is the optional end-of-run halt.
type Shutdown struct {
	Force bool          `mapstructure:"force"`
	Wait  time.Duration `mapstructure:"wait"` // default 1m
}

// Defaults returns the v1 Config used when config.toml is missing.
func Defaults() Config {
	return Config{
		Sudo: Sudo{
			Keepalive: true,
			Interval:  50 * time.Second,
		},
		Mise: Mise{
			Wrap: true,
		},
		Cache: Cache{
			Enabled: true,
		},
		Shutdown: Shutdown{
			Wait: time.Minute,
		},
	}
}
