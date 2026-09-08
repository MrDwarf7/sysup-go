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

	"sysup-go/cmd"
	"sysup-go/internal/config"
)

func TestRuntimeErrorOmitsUsage(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	dir := filepath.Join(xdg, config.AppName)
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
	root.SetArgs([]string{"list"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error")
	}
	got := buf.String()
	if strings.Contains(got, "Usage:") {
		t.Fatalf("usage dumped on runtime error:\n%s", got)
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
			root := cmd.NewRoot()
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(io.Discard)
			root.SetArgs(args)
			if err := root.Execute(); err != nil {
				t.Fatalf("Execute(%v) = %v\n%s", args, err, out.String())
			}
			path := filepath.Join(xdg, config.AppName, config.ConfigFileName)
			if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("wrote %s for %v", path, args)
			}
		})
	}
}
