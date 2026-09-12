package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/afero"

	"sysup-go/internal/config"
	"sysup-go/internal/program"
)

func (a *app) loadSpecs() ([]program.Spec, error) {
	dir, err := config.Dir()
	if err != nil {
		return nil, err
	}

	// BasePathFs is the afero equivalent of os.DirFS(dir): Load sees
	// programs.toml and programs/*.toml relative to the app dir.
	specs, err := program.Load(afero.NewBasePathFs(a.fs, dir))
	if err != nil {
		return nil, err
	}
	return program.Filter(specs, a.skip)
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
