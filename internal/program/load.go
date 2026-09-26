package program

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/afero"
)

var errNothingToRun = errors.New("nothing to run")

type fileDoc struct {
	Program []specFile `toml:"program"`
}

// specFile is the TOML shape of Spec. Enabled is a pointer so an
// omitted key stays nil and becomes true on Spec.
type specFile struct {
	Name        string   `toml:"name"`
	Enabled     *bool    `toml:"enabled"`
	Alias       string   `toml:"alias"`
	Description string   `toml:"description"`
	Optional    bool     `toml:"optional"`
	Parallel    bool     `toml:"parallel"`
	Command     []string `toml:"command"`
	Retries     Retries  `toml:"retries"`
}

func (f specFile) spec(src string) Spec {
	s := Spec{
		Name:        f.Name,
		Enabled:     true,
		Alias:       f.Alias,
		Description: f.Description,
		Optional:    f.Optional,
		Parallel:    f.Parallel,
		Command:     f.Command,
		Retries:     f.Retries,
		Source:      src,
	}
	if f.Enabled != nil {
		s.Enabled = *f.Enabled
	}
	return s
}

func Load(fsys afero.Fs) ([]Spec, error) {
	fileInfo, fileErr := fsys.Stat(FileName)
	if fileErr == nil && !fileInfo.IsDir() {
		return loadFile(fsys, FileName)
	}
	if fileErr != nil && !errors.Is(fileErr, fs.ErrNotExist) {
		return nil, &Error{Op: "read", Path: FileName, Err: fileErr}
	}

	dirInfo, dirErr := fsys.Stat(DirName)
	if dirErr == nil && dirInfo.IsDir() {
		return loadDir(fsys, DirName)
	}
	if dirErr != nil && !errors.Is(dirErr, fs.ErrNotExist) {
		return nil, &Error{Op: "read", Path: DirName, Err: dirErr}
	}
	return nil, &Error{Op: "discover", Path: FileName, Err: errNothingToRun}
}

// Filter drops specs whose Name or Alias is in skip. Unknown tokens
// return SkipError. Matching is exact and case-sensitive.
//
// Quirk: skipping a name also drops specs named name-*. Skipping
// "mirror" (or its alias) drops mirror-stage, mirror-backup, and
// mirror-swap so a split recipe is not left half-run. Skipping only
// a child does not skip the parent. The cut is parent + "-";
// "mirrors" is not a child of "mirror".
func Filter(specs []Spec, skip []string) ([]Spec, error) {
	if len(skip) == 0 {
		return specs, nil
	}
	drop, err := skipSet(specs, skip)
	if err != nil {
		return nil, err
	}
	dropped := droppedNames(specs, drop)
	out := make([]Spec, 0, len(specs))
	for _, s := range specs {
		if _, ok := dropped[s.Name]; ok {
			continue
		}
		if childOfDropped(s.Name, dropped) {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// Runnable returns specs with Enabled true. Filter does not drop
// disabled specs (list still shows them). Order preserved.
func Runnable(specs []Spec) []Spec {
	out := make([]Spec, 0, len(specs))
	for _, s := range specs {
		if s.Enabled {
			out = append(out, s)
		}
	}
	return out
}

func skipSet(specs []Spec, skip []string) (map[string]struct{}, error) {
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
	return drop, nil
}

func droppedNames(specs []Spec, drop map[string]struct{}) map[string]struct{} {
	dropped := make(map[string]struct{}, len(specs))
	for _, s := range specs {
		_, skipName := drop[s.Name]
		_, skipAlias := drop[s.Alias]
		if skipName || skipAlias {
			dropped[s.Name] = struct{}{}
		}
	}
	return dropped
}

// childOfDropped reports whether name is parent + "-" + rest for any
// skipped parent. This is the Filter name-* quirk, not filename magic.
func childOfDropped(name string, dropped map[string]struct{}) bool {
	for parent := range dropped {
		if parent != "" && strings.HasPrefix(name, parent+"-") {
			return true
		}
	}
	return false
}

func loadFile(fsys afero.Fs, src string) ([]Spec, error) {
	var doc fileDoc
	if err := decode(fsys, src, &doc); err != nil {
		return nil, err
	}
	if len(doc.Program) == 0 {
		return nil, &Error{Op: "discover", Path: src, Err: errNothingToRun}
	}
	specs := make([]Spec, 0, len(doc.Program))
	for _, raw := range doc.Program {
		s := raw.spec(src)
		if err := validate(s); err != nil {
			return nil, err
		}
		specs = append(specs, s)
	}
	if err := unique(specs); err != nil {
		return nil, err
	}
	return specs, nil
}

func loadDir(fsys afero.Fs, dir string) ([]Spec, error) {
	pattern := path.Join(dir, "*.toml")
	matches, err := afero.Glob(fsys, pattern)
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

func loadOne(fsys afero.Fs, src string) (Spec, error) {
	var raw specFile
	if err := decode(fsys, src, &raw); err != nil {
		return Spec{}, err
	}
	s := raw.spec(src)
	if err := validate(s); err != nil {
		return Spec{}, err
	}
	return s, nil
}

func decode(fsys afero.Fs, src string, v any) error {
	f, err := fsys.Open(src)
	if err != nil {
		return &Error{Op: "read", Path: src, Err: err}
	}
	defer f.Close()
	dec := toml.NewDecoder(f).DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return &Error{Op: "decode", Path: src, Err: err}
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
	if s.Retries.MaxAttempts < 0 {
		return &Error{Op: "validate", Path: s.Source, Err: errors.New("retries.max_attempts negative")}
	}
	if s.Parallel {
		return &Error{Op: "validate", Path: s.Source, Err: errors.New("parallel not implemented")}
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
