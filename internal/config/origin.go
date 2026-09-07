package config

import "github.com/spf13/viper"

type Kind uint8

const (
	KindDefault Kind = iota
	KindFile
)

func (k Kind) String() string {
	switch k {
	case KindFile:
		return "file"
	default:
		return "default"
	}
}

type Origin struct {
	Kind Kind
	Path string
}

func OriginFrom(v *viper.Viper) Origin {
	if v == nil {
		return Origin{Kind: KindDefault}
	}
	used := v.ConfigFileUsed()
	if used == "" {
		return Origin{Kind: KindDefault}
	}
	return Origin{Kind: KindFile, Path: used}
}
