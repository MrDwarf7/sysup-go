package config

import (
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

// NewViper returns a viper pointed at path on fsys. Load reads this.
// Env keys are AppName + "_" + key with "." / "-" folded to "_",
// for example SYSUP_SUDO_KEEPALIVE. AutomaticEnv is applied in Load
// after ReadInConfig so env wins over the file.
func NewViper(fsys afero.Fs, path string) *viper.Viper {
	v := viper.New()
	v.SetFs(fsys)
	v.SetConfigType(ConfigType)
	if path != "" {
		v.SetConfigFile(path)
	}
	v.SetEnvPrefix(AppName)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	return v
}
