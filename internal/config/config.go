// Package config is the TOML schema and viper decode path for the app dir.
package config

import "time"

const (
	AppName    string = "sysup"
	ConfigName string = "config"
	ConfigType string = "toml"
)

const ConfigFileName = ConfigName + "." + ConfigType

type Config struct {
	Sudo       Sudo       `mapstructure:"sudo"`
	Mise       Mise       `mapstructure:"mise"`
	Cache      Cache      `mapstructure:"cache"`
	PkgManager PkgManager `mapstructure:"pkg_manager"`
	Shutdown   Shutdown   `mapstructure:"shutdown"`
	Retries    Retries    `mapstructure:"retries"`
}

// Retries is the global retry policy. MaxAttempts 0 means no ceiling
// (recipe max applies as-is). A positive MaxAttempts is a hard cap
// on every step, even if the recipe asks for more.
type Retries struct {
	Always      bool `mapstructure:"always"`
	MaxAttempts int  `mapstructure:"max_attempts"`
}

type Sudo struct {
	Keepalive bool          `mapstructure:"keepalive"`
	Interval  time.Duration `mapstructure:"interval"`
}

type Mise struct {
	Wrap              bool `mapstructure:"wrap"`
	StripBinsFromPath bool `mapstructure:"strip_bins_from_path"`
}

type Cache struct {
	Enabled     bool     `mapstructure:"enabled"`
	IncludeDirs []string `mapstructure:"include_dirs"`
	ExcludeDirs []string `mapstructure:"exclude_dirs"`
}

type PkgManager struct {
	Name string `mapstructure:"name"`
}

type Shutdown struct {
	Force bool          `mapstructure:"force"`
	Wait  time.Duration `mapstructure:"wait"`
}

func Defaults() Config {
	return Config{
		Sudo: Sudo{
			Keepalive: true,
			Interval:  50 * time.Second,
		},
		Mise: Mise{
			Wrap:              true,
			StripBinsFromPath: true,
		},
		Cache: Cache{
			Enabled: true,
		},
		Shutdown: Shutdown{
			Wait: time.Minute,
		},
	}
}
