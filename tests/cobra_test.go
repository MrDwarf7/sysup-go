package tests

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"sysup-go/cmd"
	"sysup-go/internal/config"
)

func TestRuntimeErrorOmitsUsage(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := expectedAppDir(t, xdg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, config.ConfigFileName)
	_, existed := os.Stat(cfg)
	if err := os.WriteFile(cfg, []byte("# sysup default config.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if errors.Is(existed, fs.ErrNotExist) {
		t.Cleanup(func() {
			if err := os.Remove(cfg); err != nil && !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("cleanup %s: %v", cfg, err)
			}
		})
	}

	root := cmd.NewRoot()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"list"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error")
	}
	got := buf.String()
	if strings.Contains(got, "Usage:") {
		t.Fatalf("usage dumped on runtime error:\n%s", got)
	}
}

func TestInvalidLogLevelErrors(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := expectedAppDir(t, xdg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, config.ConfigFileName)
	if err := os.WriteFile(cfg, []byte("# sysup default config.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	root := cmd.NewRoot()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--log-level", "bogus", "list"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "log-level") {
		t.Fatalf("error = %v, want log-level", err)
	}
}

func TestUnknownFlagStillShowsUsage(t *testing.T) {
	root := cmd.NewRoot()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--bogus"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(buf.String(), "Usage:") {
		t.Fatalf("want usage on unknown flag:\n%s", buf.String())
	}
}

func TestCobraMachineryDoesNotWriteConfig(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	cases := [][]string{
		{"completion", "bash"},
		{"completion", "fish"},
		{"completion", "zsh"},
		{cobra.ShellCompRequestCmd, "list", ""},
		{"help"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			path := filepath.Join(expectedAppDir(t, xdg), config.ConfigFileName)
			_, before := os.Stat(path)
			root := cmd.NewRoot()
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(io.Discard)
			root.SetArgs(args)
			if err := root.Execute(); err != nil {
				t.Fatalf("Execute(%v) = %v\n%s", args, err, out.String())
			}
			_, after := os.Stat(path)
			if errors.Is(before, fs.ErrNotExist) && after == nil {
				t.Fatalf("wrote %s for %v", path, args)
			}
		})
	}
}

func TestListMarksDisabled(t *testing.T) {
	writeAppTree(t, map[string]string{
		"10-on.toml":  "name = \"on\"\nalias = \"o\"\ndescription = \"runs\"\ncommand = [\"true\"]\n",
		"20-off.toml": "name = \"off\"\nalias = \"x\"\ndescription = \"parked\"\nenabled = false\ncommand = [\"false\"]\n",
	})
	out, err := execRoot(t, "--log-file", "-", "list")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	want := "o  on  runs\nx  off  parked  [disabled]\n"
	if out != want {
		t.Fatalf("list output:\n%q\nwant:\n%q", out, want)
	}
}

func TestListSkipDisabled(t *testing.T) {
	writeAppTree(t, map[string]string{
		"10-on.toml":  "name = \"on\"\nalias = \"o\"\ndescription = \"runs\"\ncommand = [\"true\"]\n",
		"20-off.toml": "name = \"off\"\nalias = \"x\"\ndescription = \"parked\"\nenabled = false\ncommand = [\"false\"]\n",
	})
	out, err := execRoot(t, "--log-file", "-", "-s", "x", "list")
	if err != nil {
		t.Fatalf("list -s x: %v\n%s", err, out)
	}
	want := "o  on  runs\n"
	if out != want {
		t.Fatalf("list -s x output:\n%q\nwant:\n%q", out, want)
	}
}

func TestRunIgnoresDisabled(t *testing.T) {
	writeAppTree(t, map[string]string{
		"10-off.toml": "name = \"off\"\nenabled = false\ncommand = [\"false\"]\n",
		"20-on.toml":  "name = \"on\"\ncommand = [\"echo\", \"ran\"]\n",
	})
	out, err := execRoot(t, "--log-file", "-")
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "ran") {
		t.Fatalf("enabled step did not run:\n%s", out)
	}
}

func TestRunAllDisabled(t *testing.T) {
	writeAppTree(t, map[string]string{
		"10-off.toml": "name = \"off\"\nenabled = false\ncommand = [\"false\"]\n",
	})
	out, err := execRoot(t, "--log-file", "-")
	if err != nil {
		t.Fatalf("run all-disabled: %v\n%s", err, out)
	}
}

func execRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	viper.Reset()
	root := cmd.NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}
