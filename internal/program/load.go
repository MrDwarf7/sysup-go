package program

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"

	"github.com/pelletier/go-toml/v2"
)

var errNothingToRun = errors.New("nothing to run")

type fileDoc struct {
	Program []Spec `toml:"program"`
}

type registry uint8

const (
	registryNone registry = iota
	registryFile
	registryDir
)

func Load(fsys fs.FS) ([]Spec, error) {
	filePath, dirPath := FileName, DirName
	fileInfo, fileErr := fs.Stat(fsys, filePath)
	dirInfo, dirErr := fs.Stat(fsys, dirPath)

	kind, classErr := classify(fileInfo, fileErr, dirInfo, dirErr)
	if classErr != nil {
		path := filePath
		if fileErr == nil || errors.Is(fileErr, fs.ErrNotExist) {
			path = dirPath
		}
		return nil, &Error{Op: "read", Path: path, Err: classErr}
	}

	switch kind {
	case registryFile:
		return loadFile(fsys, filePath)
	case registryDir:
		return loadDir(fsys, dirPath)
	default:
		return nil, &Error{Op: "discover", Path: filePath, Err: errNothingToRun}
	}
}

func classify(file fs.FileInfo, fileErr error, dir fs.FileInfo, dirErr error) (registry, error) {
	if fileErr == nil && !file.IsDir() {
		return registryFile, nil
	}
	if dirErr == nil && dir.IsDir() {
		return registryDir, nil
	}
	if fileErr != nil && !errors.Is(fileErr, fs.ErrNotExist) {
		return registryNone, fileErr
	}
	if dirErr != nil && !errors.Is(dirErr, fs.ErrNotExist) {
		return registryNone, dirErr
	}
	return registryNone, nil
}

func Filter(specs []Spec, skip []string) ([]Spec, error) {
	if len(skip) == 0 {
		return specs, nil
	}

	known := make(map[string]struct{}, len(specs)*2)
	for _, s := range specs {
		known[s.Name] = struct{}{}
		known[s.Alias] = struct{}{}
	}
	delete(known, "")

	drop := make(map[string]struct{}, len(skip))
	for _, token := range skip {
		if _, ok := known[token]; !ok {
			return nil, &SkipError{Token: token}
		}
		drop[token] = struct{}{}
	}

	out := make([]Spec, 0, len(specs))
	for _, s := range specs {
		_, skipName := drop[s.Name]
		_, skipAlias := drop[s.Alias]
		if skipName || skipAlias {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

func loadFile(fsys fs.FS, path string) ([]Spec, error) {
	var doc fileDoc
	if err := decode(fsys, path, &doc); err != nil {
		return nil, err
	}
	if len(doc.Program) == 0 {
		return nil, &Error{Op: "discover", Path: path, Err: errNothingToRun}
	}
	for i := range doc.Program {
		doc.Program[i].Source = path
	}
	for _, s := range doc.Program {
		if err := validate(s); err != nil {
			return nil, err
		}
	}
	if err := unique(doc.Program); err != nil {
		return nil, err
	}
	return doc.Program, nil
}

func loadDir(fsys fs.FS, dir string) ([]Spec, error) {
	matches, err := fs.Glob(fsys, dir+"/*"+filepath.Ext(FileName))
	if err != nil {
		return nil, &Error{Op: "discover", Path: dir, Err: err}
	}
	slices.Sort(matches)
	if len(matches) == 0 {
		return nil, &Error{Op: "discover", Path: dir, Err: errNothingToRun}
	}

	specs := make([]Spec, 0, len(matches))
	for _, path := range matches {
		spec, err := loadOne(fsys, path)
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

func loadOne(fsys fs.FS, path string) (Spec, error) {
	var spec Spec
	if err := decode(fsys, path, &spec); err != nil {
		return Spec{}, err
	}
	spec.Source = path
	if err := validate(spec); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

func decode(fsys fs.FS, path string, v any) error {
	f, err := fsys.Open(path)
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
	for _, s := range specs {
		if prev, ok := names[s.Name]; ok {
			return &Error{
				Op:   "validate",
				Path: s.Source,
				Err:  fmt.Errorf("duplicate name %q: %s and %s", s.Name, prev, s.Source),
			}
		}
		names[s.Name] = s.Source
	}

	aliases := make(map[string]string, len(specs))
	for _, s := range specs {
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
