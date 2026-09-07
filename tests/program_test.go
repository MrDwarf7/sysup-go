package tests

import (
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"sysup-go/internal/program"
)

func TestProgramLoadNeitherFileNorDir(t *testing.T) {
	t.Parallel()
	_, err := program.Load(fstest.MapFS{})
	assertNothingToRun(t, err)
}

func TestProgramLoadEmptyFile(t *testing.T) {
	t.Parallel()
	_, err := program.Load(fstest.MapFS{
		program.FileName: {Data: []byte("")},
	})
	assertNothingToRun(t, err)
}

func TestProgramLoadEmptyDir(t *testing.T) {
	t.Parallel()
	_, err := program.Load(fstest.MapFS{
		program.DirName: {Mode: fs.ModeDir},
	})
	assertNothingToRun(t, err)
}

func TestProgramLoadFileWinsOverDir(t *testing.T) {
	t.Parallel()
	specs, err := program.Load(fstest.MapFS{
		program.FileName: {Data: []byte(`
[[program]]
name = "from-file"
alias = "f"
description = "file"
command = ["echo", "file"]
`)},
		program.DirName + "/foo.toml": {Data: []byte(`
name = "from-dir"
alias = "d"
description = "dir"
command = ["echo", "dir"]
`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := specNames(specs); !slices.Equal(got, []string{"from-file"}) {
		t.Errorf("names = %v, want [from-file]", got)
	}
	if specs[0].Source != program.FileName {
		t.Errorf("Source = %q, want %s", specs[0].Source, program.FileName)
	}
}

func TestProgramLoadDirLexicalOrder(t *testing.T) {
	t.Parallel()
	specs, err := program.Load(fstest.MapFS{
		program.DirName + "/20-b.toml": {Data: []byte("name = \"b\"\ncommand = [\"echo\", \"b\"]\n")},
		program.DirName + "/10-a.toml": {Data: []byte("name = \"a\"\ncommand = [\"echo\", \"a\"]\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := specNames(specs); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("names = %v, want [a b]", got)
	}
}

func TestProgramLoadDirLexicalNotNumeric(t *testing.T) {
	t.Parallel()
	specs, err := program.Load(fstest.MapFS{
		program.DirName + "/9-z.toml":  {Data: []byte("name = \"z\"\ncommand = [\"echo\", \"z\"]\n")},
		program.DirName + "/10-a.toml": {Data: []byte("name = \"a\"\ncommand = [\"echo\", \"a\"]\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := specNames(specs); !slices.Equal(got, []string{"a", "z"}) {
		t.Errorf("names = %v, want [a z]", got)
	}
}

func TestProgramLoadDuplicateNames(t *testing.T) {
	t.Parallel()
	pathA := program.DirName + "/10-a.toml"
	pathB := program.DirName + "/20-b.toml"
	_, err := program.Load(fstest.MapFS{
		pathA: {Data: []byte("name = \"pacman\"\ncommand = [\"echo\", \"a\"]\n")},
		pathB: {Data: []byte("name = \"pacman\"\ncommand = [\"echo\", \"b\"]\n")},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, pathA) || !strings.Contains(msg, pathB) {
		t.Errorf("Error() = %q, want both paths", msg)
	}
}

func TestProgramLoadDuplicateAliases(t *testing.T) {
	t.Parallel()
	pathA := program.DirName + "/10-a.toml"
	pathB := program.DirName + "/20-b.toml"
	_, err := program.Load(fstest.MapFS{
		pathA: {Data: []byte("name = \"mirror\"\nalias = \"p\"\ncommand = [\"echo\", \"a\"]\n")},
		pathB: {Data: []byte("name = \"pacman\"\nalias = \"p\"\ncommand = [\"echo\", \"b\"]\n")},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, pathA) || !strings.Contains(msg, pathB) {
		t.Errorf("Error() = %q, want both paths", msg)
	}
}

func TestProgramLoadMissingCommand(t *testing.T) {
	t.Parallel()
	_, err := program.Load(fstest.MapFS{
		program.DirName + "/10-a.toml": {Data: []byte("name = \"pacman\"\nalias = \"p\"\ndescription = \"pkgs\"\n")},
	})
	var pe *program.Error
	if !errors.As(err, &pe) {
		t.Fatalf("got %T %v, want *program.Error", err, err)
	}
	if pe.Op != "validate" {
		t.Errorf("Op = %q, want validate", pe.Op)
	}
}

func TestProgramLoadUnknownFieldKind(t *testing.T) {
	t.Parallel()
	_, err := program.Load(fstest.MapFS{
		program.DirName + "/10-a.toml": {Data: []byte("name = \"pacman\"\nkind = \"exec\"\ncommand = [\"sudo\"]\n")},
	})
	var pe *program.Error
	if !errors.As(err, &pe) {
		t.Fatalf("got %T %v, want *program.Error", err, err)
	}
	if pe.Op != "decode" {
		t.Errorf("Op = %q, want decode", pe.Op)
	}
}

func TestProgramLoadFileArrayOrder(t *testing.T) {
	t.Parallel()
	specs, err := program.Load(fstest.MapFS{
		program.FileName: {Data: []byte(`
[[program]]
name = "mirror"
alias = "m"
description = "mirrors"
command = ["rate-mirrors"]

[[program]]
name = "pacman"
alias = "p"
description = "pkgs"
command = ["pacman", "-Syu"]
`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := specNames(specs); !slices.Equal(got, []string{"mirror", "pacman"}) {
		t.Errorf("names = %v, want [mirror pacman]", got)
	}
	if specs[0].Optional || specs[0].Parallel {
		t.Errorf("optional/parallel = %v/%v, want false/false", specs[0].Optional, specs[0].Parallel)
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
