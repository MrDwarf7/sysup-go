package program

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadNeitherFileNorDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := Load(dir)
	assertNothingToRun(t, err)
}

func TestLoadEmptyProgramsFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "programs.toml"), "")
	_, err := Load(dir)
	assertNothingToRun(t, err)
}

func TestLoadEmptyProgramsDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "programs"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	assertNothingToRun(t, err)
}

func TestLoadFileWinsOverDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "programs.toml"), `
[[program]]
name = "from-file"
alias = "f"
description = "file"
command = ["echo", "file"]
`)
	mustWrite(t, filepath.Join(dir, "programs", "foo.toml"), `
name = "from-dir"
alias = "d"
description = "dir"
command = ["echo", "dir"]
`)
	specs, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := specNames(specs)
	want := []string{"from-file"}
	if !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v", got, want)
	}
	if filepath.Base(specs[0].Source) != "programs.toml" {
		t.Errorf("Source = %q, want programs.toml", specs[0].Source)
	}
}

func TestLoadDirLexicalOrder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "programs", "20-b.toml"), `
name = "b"
command = ["echo", "b"]
`)
	mustWrite(t, filepath.Join(dir, "programs", "10-a.toml"), `
name = "a"
command = ["echo", "a"]
`)
	specs, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := specNames(specs)
	want := []string{"a", "b"}
	if !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v", got, want)
	}
}

func TestLoadDirLexicalNotNumeric(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "programs", "9-z.toml"), `
name = "z"
command = ["echo", "z"]
`)
	mustWrite(t, filepath.Join(dir, "programs", "10-a.toml"), `
name = "a"
command = ["echo", "a"]
`)
	specs, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := specNames(specs)
	want := []string{"a", "z"}
	if !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v", got, want)
	}
}

func TestLoadDuplicateNames(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pathA := filepath.Join(dir, "programs", "10-a.toml")
	pathB := filepath.Join(dir, "programs", "20-b.toml")
	mustWrite(t, pathA, `
name = "pacman"
command = ["echo", "a"]
`)
	mustWrite(t, pathB, `
name = "pacman"
command = ["echo", "b"]
`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, pathA) || !strings.Contains(msg, pathB) {
		t.Errorf("Error() = %q, want both %q and %q", msg, pathA, pathB)
	}
}

func TestLoadDuplicateAliases(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pathA := filepath.Join(dir, "programs", "10-a.toml")
	pathB := filepath.Join(dir, "programs", "20-b.toml")
	mustWrite(t, pathA, `
name = "mirror"
alias = "p"
command = ["echo", "a"]
`)
	mustWrite(t, pathB, `
name = "pacman"
alias = "p"
command = ["echo", "b"]
`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, pathA) || !strings.Contains(msg, pathB) {
		t.Errorf("Error() = %q, want both %q and %q", msg, pathA, pathB)
	}
}

func TestLoadMissingCommand(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "programs", "10-a.toml"), `
name = "pacman"
alias = "p"
description = "pkgs"
`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error")
	}
	var pe *Error
	if !errors.As(err, &pe) {
		t.Fatalf("got %T %v, want *Error", err, err)
	}
	if pe.Op != "validate" {
		t.Errorf("Op = %q, want validate", pe.Op)
	}
	if !strings.Contains(err.Error(), "command") {
		t.Errorf("Error() = %q, want command", err.Error())
	}
}

func TestLoadUnknownFieldKind(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "programs", "10-a.toml"), `
name = "pacman"
kind = "exec"
command = ["sudo", "pacman", "-Syyu"]
`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error")
	}
	var pe *Error
	if !errors.As(err, &pe) {
		t.Fatalf("got %T %v, want *Error", err, err)
	}
	if pe.Op != "decode" {
		t.Errorf("Op = %q, want decode", pe.Op)
	}
}

func TestLoadProgramsFileArrayOrder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "programs.toml"), `
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
`)
	specs, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := specNames(specs)
	want := []string{"mirror", "pacman"}
	if !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v", got, want)
	}
	if specs[0].Optional || specs[0].Parallel {
		t.Errorf("optional/parallel = %v/%v, want false/false", specs[0].Optional, specs[0].Parallel)
	}
	if !slices.Equal(specs[0].Command, []string{"rate-mirrors"}) {
		t.Errorf("command = %v, want [rate-mirrors]", specs[0].Command)
	}
	if specs[1].Alias != "p" {
		t.Errorf("alias = %q, want p", specs[1].Alias)
	}
}

func TestFilter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "programs.toml"), `
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

[[program]]
name = "aur"
alias = "a"
description = "aur"
command = ["paru"]
`)
	specs, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("by name and alias", func(t *testing.T) {
		got, err := Filter(specs, []string{"pacman", "m"})
		if err != nil {
			t.Fatal(err)
		}
		if names := specNames(got); !slices.Equal(names, []string{"aur"}) {
			t.Errorf("names = %v, want [aur]", names)
		}
	})

	t.Run("unknown token", func(t *testing.T) {
		_, err := Filter(specs, []string{"nope"})
		var se *SkipError
		if !errors.As(err, &se) {
			t.Fatalf("got %T %v, want *SkipError", err, err)
		}
		if se.Token != "nope" {
			t.Errorf("Token = %q, want nope", se.Token)
		}
	})

	t.Run("duplicate skip tokens", func(t *testing.T) {
		got, err := Filter(specs, []string{"p", "p"})
		if err != nil {
			t.Fatal(err)
		}
		if names := specNames(got); !slices.Equal(names, []string{"mirror", "aur"}) {
			t.Errorf("names = %v, want [mirror aur]", names)
		}
	})
}

func assertNothingToRun(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	var pe *Error
	if !errors.As(err, &pe) {
		t.Fatalf("got %T %v, want *Error", err, err)
	}
	if pe.Op != "discover" {
		t.Errorf("Op = %q, want discover", pe.Op)
	}
	if !strings.Contains(err.Error(), "nothing to run") {
		t.Errorf("Error() = %q, want nothing to run", err.Error())
	}
}

func specNames(specs []Spec) []string {
	names := make([]string, len(specs))
	for i, s := range specs {
		names[i] = s.Name
	}
	return names
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
