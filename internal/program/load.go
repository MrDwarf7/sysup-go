package program

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/pelletier/go-toml/v2"
)

var errNothingToRun = errors.New("nothing to run")

type fileDoc struct {
	Program []Spec `toml:"program"`
}

// Load discovers programs from dir/programs.toml or dir/programs/*.toml.
func Load(dir string) ([]Spec, error) {
	filePath := filepath.Join(dir, "programs.toml")
	st, err := os.Stat(filePath)
	switch {
	case err == nil && !st.IsDir():
		return loadFile(filePath)
	case err == nil:
		// A directory named programs.toml is not the file registry.
	case !errors.Is(err, os.ErrNotExist):
		return nil, &Error{Op: "read", Path: filePath, Err: err}
	}

	dirPath := filepath.Join(dir, "programs")
	st, err = os.Stat(dirPath)
	switch {
	case err == nil && st.IsDir():
		return loadDir(dirPath)
	case err == nil:
		// A file named programs is not the directory registry.
	case !errors.Is(err, os.ErrNotExist):
		return nil, &Error{Op: "read", Path: dirPath, Err: err}
	}

	return nil, &Error{Op: "discover", Path: dir, Err: errNothingToRun}
}

// Filter drops specs whose Name or Alias is in skip. Remaining order is kept.
func Filter(specs []Spec, skip []string) ([]Spec, error) {
	if len(skip) == 0 {
		return specs, nil
	}

	tokens := make(map[string]struct{}, len(specs)*2)
	for _, s := range specs {
		tokens[s.Name] = struct{}{}
		if s.Alias != "" {
			tokens[s.Alias] = struct{}{}
		}
	}
	for _, token := range skip {
		if _, ok := tokens[token]; !ok {
			return nil, &SkipError{Token: token}
		}
	}

	drop := make(map[string]struct{}, len(skip))
	for _, token := range skip {
		drop[token] = struct{}{}
	}

	out := make([]Spec, 0, len(specs))
	for _, s := range specs {
		if _, ok := drop[s.Name]; ok {
			continue
		}
		if s.Alias != "" {
			if _, ok := drop[s.Alias]; ok {
				continue
			}
		}
		out = append(out, s)
	}
	return out, nil
}

func loadFile(path string) ([]Spec, error) {
	var doc fileDoc
	if err := decode(path, &doc); err != nil {
		return nil, err
	}
	if len(doc.Program) == 0 {
		return nil, &Error{Op: "discover", Path: path, Err: errNothingToRun}
	}
	for i := range doc.Program {
		doc.Program[i].Source = path
		if err := validate(doc.Program[i]); err != nil {
			return nil, err
		}
	}
	if err := unique(doc.Program); err != nil {
		return nil, err
	}
	return doc.Program, nil
}

func loadDir(dir string) ([]Spec, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.toml"))
	if err != nil {
		return nil, &Error{Op: "discover", Path: dir, Err: err}
	}
	names := make([]string, len(matches))
	for i, m := range matches {
		names[i] = filepath.Base(m)
	}
	slices.Sort(names)
	if len(names) == 0 {
		return nil, &Error{Op: "discover", Path: dir, Err: errNothingToRun}
	}

	specs := make([]Spec, 0, len(names))
	for _, name := range names {
		spec, err := loadOne(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}
	if err := unique(specs); err != nil {
		return nil, err
	}
	return specs, nil
}

func loadOne(path string) (Spec, error) {
	var spec Spec
	if err := decode(path, &spec); err != nil {
		return Spec{}, err
	}
	spec.Source = path
	if err := validate(spec); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

func decode(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return &Error{Op: "read", Path: path, Err: err}
	}
	defer f.Close()
	dec := toml.NewDecoder(f).DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return &Error{Op: "decode", Path: path, Err: err}
	}
	return nil
}

func validate(s Spec) error {
	if s.Name == "" {
		return &Error{Op: "validate", Path: s.Source, Err: errors.New("name required")}
	}
	if len(s.Command) < 1 {
		return &Error{Op: "validate", Path: s.Source, Err: errors.New("command required")}
	}
	for i, arg := range s.Command {
		if arg == "" {
			return &Error{Op: "validate", Path: s.Source, Err: fmt.Errorf("command[%d] empty", i)}
		}
	}
	return nil
}

func unique(specs []Spec) error {
	names := make(map[string]string, len(specs))
	aliases := make(map[string]string, len(specs))
	for _, s := range specs {
		if prev, ok := names[s.Name]; ok {
			return &Error{
				Op:   "validate",
				Path: s.Source,
				Err:  fmt.Errorf("duplicate name %q: %s and %s", s.Name, prev, s.Source),
			}
		}
		names[s.Name] = s.Source
		if s.Alias == "" {
			continue
		}
		if prev, ok := aliases[s.Alias]; ok {
			return &Error{
				Op:   "validate",
				Path: s.Source,
				Err:  fmt.Errorf("duplicate alias %q: %s and %s", s.Alias, prev, s.Source),
			}
		}
		aliases[s.Alias] = s.Source
	}
	return nil
}
