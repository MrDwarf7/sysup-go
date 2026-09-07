package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/viper"

	"sysup-go/internal/config"
	"sysup-go/internal/program"
)

func loadSpecs() ([]program.Spec, error) {
	dir, err := config.EnsureAppDir(afero.NewOsFs())
	if err != nil {
		return nil, err
	}
	specs, err := program.Load(os.DirFS(dir))
	if err != nil {
		return nil, err
	}
	return program.Filter(specs, viper.GetStringSlice("skip"))
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
