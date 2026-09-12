package tests

import (
	"errors"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"sysup-go/internal/program"
)

func TestProgramLoadNeitherFileNorDir(t *testing.T) {
	t.Parallel()
	_, err := program.Load(afero.NewMemMapFs())
	assertNothingToRun(t, err)
}

func TestProgramLoadEmptyFile(t *testing.T) {
	t.Parallel()
	for _, backend := range ioBackends() {
		t.Run(backend, func(t *testing.T) {
			t.Parallel()
			_, err := loadProgramTree(t, "empty-file", backend)
			assertNothingToRun(t, err)
		})
	}
}

func TestProgramLoadEmptyDir(t *testing.T) {
	t.Parallel()
	for _, backend := range ioBackends() {
		t.Run(backend, func(t *testing.T) {
			t.Parallel()
			_, err := loadProgramTree(t, "empty-dir", backend)
			assertNothingToRun(t, err)
		})
	}
}

func TestProgramLoadFileWinsOverDir(t *testing.T) {
	t.Parallel()
	for _, backend := range ioBackends() {
		t.Run(backend, func(t *testing.T) {
			t.Parallel()
			specs, err := loadProgramTree(t, "file-wins", backend)
			if err != nil {
				t.Fatal(err)
			}
			if got := specNames(specs); !slices.Equal(got, []string{"from-file"}) {
				t.Errorf("names = %v, want [from-file]", got)
			}
			if specs[0].Source != program.FileName {
				t.Errorf("Source = %q, want %s", specs[0].Source, program.FileName)
			}
		})
	}
}

func TestProgramLoadDirLexicalOrder(t *testing.T) {
	t.Parallel()
	for _, backend := range ioBackends() {
		t.Run(backend, func(t *testing.T) {
			t.Parallel()
			specs, err := loadProgramTree(t, "lexical", backend)
			if err != nil {
				t.Fatal(err)
			}
			if got := specNames(specs); !slices.Equal(got, []string{"a", "b"}) {
				t.Errorf("names = %v, want [a b]", got)
			}
		})
	}
}

func TestProgramLoadDirLexicalNotNumeric(t *testing.T) {
	t.Parallel()
	for _, backend := range ioBackends() {
		t.Run(backend, func(t *testing.T) {
			t.Parallel()
			specs, err := loadProgramTree(t, "lexical-not-numeric", backend)
			if err != nil {
				t.Fatal(err)
			}
			if got := specNames(specs); !slices.Equal(got, []string{"a", "z"}) {
				t.Errorf("names = %v, want [a z]", got)
			}
		})
	}
}

func TestProgramLoadDuplicateNames(t *testing.T) {
	t.Parallel()
	pathA := program.DirName + "/10-a.toml"
	pathB := program.DirName + "/20-b.toml"
	for _, backend := range ioBackends() {
		t.Run(backend, func(t *testing.T) {
			t.Parallel()
			_, err := loadProgramTree(t, "dup-names", backend)
			if err == nil {
				t.Fatal("expected error")
			}
			msg := err.Error()
			if !strings.Contains(msg, pathA) || !strings.Contains(msg, pathB) {
				t.Errorf("Error() = %q, want both paths", msg)
			}
		})
	}
}

func TestProgramLoadDuplicateAliases(t *testing.T) {
	t.Parallel()
	pathA := program.DirName + "/10-a.toml"
	pathB := program.DirName + "/20-b.toml"
	for _, backend := range ioBackends() {
		t.Run(backend, func(t *testing.T) {
			t.Parallel()
			_, err := loadProgramTree(t, "dup-aliases", backend)
			if err == nil {
				t.Fatal("expected error")
			}
			msg := err.Error()
			if !strings.Contains(msg, pathA) || !strings.Contains(msg, pathB) {
				t.Errorf("Error() = %q, want both paths", msg)
			}
		})
	}
}

func TestProgramLoadMissingCommand(t *testing.T) {
	t.Parallel()
	for _, backend := range ioBackends() {
		t.Run(backend, func(t *testing.T) {
			t.Parallel()
			_, err := loadProgramTree(t, "missing-command", backend)
			var pe *program.Error
			if !errors.As(err, &pe) {
				t.Fatalf("got %T %v, want *program.Error", err, err)
			}
			if pe.Op != "validate" {
				t.Errorf("Op = %q, want validate", pe.Op)
			}
		})
	}
}

func TestProgramLoadUnknownFieldKind(t *testing.T) {
	t.Parallel()
	for _, backend := range ioBackends() {
		t.Run(backend, func(t *testing.T) {
			t.Parallel()
			_, err := loadProgramTree(t, "unknown-kind", backend)
			var pe *program.Error
			if !errors.As(err, &pe) {
				t.Fatalf("got %T %v, want *program.Error", err, err)
			}
			if pe.Op != "decode" {
				t.Errorf("Op = %q, want decode", pe.Op)
			}
		})
	}
}

func TestProgramLoadFileArrayOrder(t *testing.T) {
	t.Parallel()
	for _, backend := range ioBackends() {
		t.Run(backend, func(t *testing.T) {
			t.Parallel()
			specs, err := loadProgramTree(t, "file-array", backend)
			if err != nil {
				t.Fatal(err)
			}
			if got := specNames(specs); !slices.Equal(got, []string{"mirror", "pacman"}) {
				t.Errorf("names = %v, want [mirror pacman]", got)
			}
			if specs[0].Optional || specs[0].Parallel {
				t.Errorf("optional/parallel = %v/%v, want false/false", specs[0].Optional, specs[0].Parallel)
			}
		})
	}
}

func TestProgramFilter(t *testing.T) {
	t.Parallel()
	specs := []program.Spec{
		{Name: "mirror", Alias: "m", Command: []string{"rate-mirrors"}},
		{Name: "pacman", Alias: "p", Command: []string{"pacman"}},
		{Name: "aur", Alias: "a", Command: []string{"paru"}},
	}

	t.Run("by name and alias", func(t *testing.T) {
		got, err := program.Filter(specs, []string{"pacman", "m"})
		if err != nil {
			t.Fatal(err)
		}
		if names := specNames(got); !slices.Equal(names, []string{"aur"}) {
			t.Errorf("names = %v, want [aur]", names)
		}
	})

	t.Run("unknown token", func(t *testing.T) {
		_, err := program.Filter(specs, []string{"nope"})
		var se *program.SkipError
		if !errors.As(err, &se) {
			t.Fatalf("got %T %v, want *program.SkipError", err, err)
		}
		if se.Token != "nope" {
			t.Errorf("Token = %q, want nope", se.Token)
		}
	})

	t.Run("duplicate skip tokens", func(t *testing.T) {
		got, err := program.Filter(specs, []string{"p", "p"})
		if err != nil {
			t.Fatal(err)
		}
		if names := specNames(got); !slices.Equal(names, []string{"mirror", "aur"}) {
			t.Errorf("names = %v, want [mirror aur]", names)
		}
	})

	t.Run("skip parent drops name- prefix children", func(t *testing.T) {
		chain := []program.Spec{
			{Name: "mirror", Alias: "m", Command: []string{"rate-mirrors"}},
			{Name: "mirror-stage", Alias: "mt", Command: []string{"install"}},
			{Name: "mirror-backup", Alias: "mb", Command: []string{"cp"}},
			{Name: "mirror-swap", Alias: "mw", Command: []string{"mv"}},
			{Name: "pacman", Alias: "p", Command: []string{"pacman"}},
		}
		got, err := program.Filter(chain, []string{"m"})
		if err != nil {
			t.Fatal(err)
		}
		if names := specNames(got); !slices.Equal(names, []string{"pacman"}) {
			t.Errorf("names = %v, want [pacman]", names)
		}
		got, err = program.Filter(chain, []string{"mt"})
		if err != nil {
			t.Fatal(err)
		}
		if names := specNames(got); !slices.Equal(names, []string{"mirror", "mirror-backup", "mirror-swap", "pacman"}) {
			t.Errorf("skip child only = %v", names)
		}
	})

	t.Run("empty alias is not a skip token", func(t *testing.T) {
		withBlank := []program.Spec{
			{Name: "mirror", Command: []string{"rate-mirrors"}},
			{Name: "pacman", Alias: "p", Command: []string{"pacman"}},
		}
		_, err := program.Filter(withBlank, []string{""})
		var se *program.SkipError
		if !errors.As(err, &se) {
			t.Fatalf("got %T %v, want *program.SkipError", err, err)
		}
		if se.Token != "" {
			t.Errorf("Token = %q, want empty", se.Token)
		}
		got, err := program.Filter(withBlank, []string{"p"})
		if err != nil {
			t.Fatal(err)
		}
		if names := specNames(got); !slices.Equal(names, []string{"mirror"}) {
			t.Errorf("names = %v, want [mirror]", names)
		}
	})
}

func TestExampleProgramsLoad(t *testing.T) {
	t.Parallel()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "examples")
	specs, err := program.Load(afero.NewBasePathFs(afero.NewOsFs(), root))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"mirror", "mirror-stage", "mirror-backup", "mirror-swap", "pacman", "aur", "rustup", "neovim", "ya", "hermes"}
	if got := specNames(specs); !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v", got, want)
	}
}

func assertNothingToRun(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	var pe *program.Error
	if !errors.As(err, &pe) {
		t.Fatalf("got %T %v, want *program.Error", err, err)
	}
	if pe.Op != "discover" {
		t.Errorf("Op = %q, want discover", pe.Op)
	}
	if !strings.Contains(err.Error(), "nothing to run") {
		t.Errorf("Error() = %q, want nothing to run", err.Error())
	}
}
