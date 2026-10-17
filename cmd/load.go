package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/afero"

	"sysup-go/internal/config"
	"sysup-go/internal/logfmt"
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
	color := false
	if f, ok := w.(*os.File); ok {
		color = logfmt.ColorTTY(f)
	}
	for _, s := range specs {
		alias := s.Alias
		if alias == "" {
			alias = "-"
		}
		line := fmt.Sprintf("%s  %s  %s", alias, s.Name, s.Description)
		if !s.Enabled {
			line += "  [disabled]"
			if color {
				line = "\x1b[90m" + line + "\x1b[0m"
			}
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}
