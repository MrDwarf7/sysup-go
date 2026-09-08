package config

import (
	"errors"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

func decodeOpts(strict bool) viper.DecoderConfigOption {
	return func(dc *mapstructure.DecoderConfig) {
		dc.ZeroFields = false
		dc.DecodeHook = mapstructure.StringToTimeDurationHookFunc()
		dc.ErrorUnused = strict
	}
}

func unmarshal(v *viper.Viper, strict bool) (Config, error) {
	cfg := Defaults()
	if v == nil {
		return cfg, nil
	}
	if err := v.Unmarshal(&cfg, decodeOpts(strict)); err != nil {
		return Config{}, &Error{Op: "decode", Path: v.ConfigFileUsed(), Err: err}
	}
	return cfg, nil
}

func Unmarshal(v *viper.Viper) (Config, error) {
	return unmarshal(v, false)
}

func UnmarshalStrict(v *viper.Viper) (Config, error) {
	return unmarshal(v, true)
}

func Load(v *viper.Viper) (Config, error) {
	if v == nil {
		return Defaults(), nil
	}
	err := v.ReadInConfig()
	v.AutomaticEnv()
	if err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			return Unmarshal(v)
		}
		path := v.ConfigFileUsed()
		if path == "" {
			path = ConfigFileName
		}
		return Config{}, wrapRead(path, err)
	}
	return UnmarshalStrict(v)
}

func wrapRead(path string, err error) error {
	op := "read"
	if _, ok := errors.AsType[viper.ConfigParseError](err); ok {
		op = "decode"
	}
	return &Error{Op: op, Path: path, Err: err}
}
