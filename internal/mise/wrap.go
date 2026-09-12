// Package mise strips mise-managed bins from PATH for child processes.
// Fish used deactivate/activate because it was a shell. We are not:
// walk PATH once, drop shim + installs entries, restore the saved string.
package mise

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type LookPath func(file string) (string, error)

// State is the result of one Begin. Restore uses PathOrig only; it
// does not re-parse PATH.
type State struct {
	DidStrip bool
	PathOrig string
	PathNow  string
	ShimDir  string
	MisePath string
}

type Wrap struct {
	LookPath LookPath
	Log      *slog.Logger
}

func dataDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "mise")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "mise")
}

func shimDir() string {
	if d := os.Getenv("MISE_SHIMS_DIR"); d != "" {
		return d
	}
	return filepath.Join(dataDir(), "shims")
}

func miseManaged(entry, shim, data string) bool {
	if entry == "" {
		return false
	}
	clean := filepath.Clean(entry)
	if shim != "" && clean == filepath.Clean(shim) {
		return true
	}
	if data == "" {
		return false
	}
	data = filepath.Clean(data)
	return clean == data || strings.HasPrefix(clean, data+string(os.PathSeparator))
}

func stripPath(path, shim, data string) (string, bool) {
	parts := filepath.SplitList(path)
	keep := make([]string, 0, len(parts))
	dropped := false
	for _, p := range parts {
		if miseManaged(p, shim, data) {
			dropped = true
			continue
		}
		keep = append(keep, p)
	}
	if !dropped {
		return path, false
	}
	return strings.Join(keep, string(os.PathListSeparator)), true
}

// Begin strips mise bins from this process PATH when enabled and mise
// is present. Strip covers the shims dir and anything under the mise
// data dir (installs/*/bin), not only shims -- makepkg was picking
// ~/.xdg/data/mise/installs/python/latest/bin/python.
func (w Wrap) Begin(enabled bool) State {
	if !enabled {
		return State{}
	}
	look := w.LookPath
	if look == nil {
		look = exec.LookPath
	}
	misePath, err := look("mise")
	if err != nil {
		if w.Log != nil {
			w.Log.Info("mise not on PATH, skip wrap")
		}
		return State{}
	}
	pathOrig := os.Getenv("PATH")
	shim := shimDir()
	data := dataDir()
	stripped, did := stripPath(pathOrig, shim, data)
	if !did {
		return State{MisePath: misePath}
	}
	if err := os.Setenv("PATH", stripped); err != nil {
		if w.Log != nil {
			w.Log.Warn("mise: set PATH", "err", err)
		}
		return State{MisePath: misePath}
	}
	if w.Log != nil {
		w.Log.Info("stripped mise bins from PATH", "shim", shim)
	}
	return State{
		DidStrip: true,
		PathOrig: pathOrig,
		PathNow:  stripped,
		ShimDir:  shim,
		MisePath: misePath,
	}
}

func (s State) RestoreProcessPath() error {
	if !s.DidStrip {
		return nil
	}
	if err := os.Setenv("PATH", s.PathOrig); err != nil {
		return &Error{Op: "restore", Err: err}
	}
	return nil
}

// ChildEnv copies base and replaces PATH with the stripped value.
func (s State) ChildEnv(base []string) []string {
	if !s.DidStrip {
		return base
	}
	out := make([]string, 0, len(base))
	found := false
	for _, kv := range base {
		if strings.HasPrefix(kv, "PATH=") {
			out = append(out, "PATH="+s.PathNow)
			found = true
			continue
		}
		out = append(out, kv)
	}
	if !found {
		out = append(out, "PATH="+s.PathNow)
	}
	return out
}

// Up runs `mise up` with the binary resolved before PATH was stripped.
func (s State) Up(ctx context.Context) error {
	if !s.DidStrip || s.MisePath == "" {
		return nil
	}
	cmd := exec.CommandContext(ctx, "mise", "up")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return &Error{Op: "up", Err: err}
	}
	return nil
}
