package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"sysup-go/internal/config"
	"sysup-go/internal/program"
)

func appDir() (string, error) {
	if cfgFile != "" {
		return filepath.Dir(cfgFile), nil
	}
	return config.Dir()
}

func loadSpecs() ([]program.Spec, error) {
	dir, err := appDir()
	if err != nil {
		return nil, err
	}
	specs, err := program.Load(dir)
	if err != nil {
		return nil, err
	}
	return program.Filter(specs, skip)
}

func configLabel() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "config.toml")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "defaults", nil
		}
		return "", err
	}
	return path, nil
}

func printList(w io.Writer, specs []program.Spec) error {
	for _, s := range specs {
		alias := s.Alias
		if alias == "" {
			alias = "-"
		}
		if _, err := fmt.Fprintf(w, "%s  %s  %s\n", alias, s.Name, s.Description); err != nil {
			return err
		}
	}
	return nil
}
