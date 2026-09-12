package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sysup-go/internal/mise"
)

func TestMiseBeginDisabled(t *testing.T) {
	orig := os.Getenv("PATH")
	w := mise.Wrap{LookPath: func(string) (string, error) { t.Fatal("LookPath"); return "", nil }}
	st := w.Begin(false)
	if st.DidStrip {
		t.Fatal("DidStrip")
	}
	if os.Getenv("PATH") != orig {
		t.Fatal("PATH changed")
	}
}

func TestMiseBeginMissing(t *testing.T) {
	orig := os.Getenv("PATH")
	w := mise.Wrap{LookPath: func(string) (string, error) { return "", os.ErrNotExist }}
	st := w.Begin(true)
	if st.DidStrip {
		t.Fatal("DidStrip")
	}
	if os.Getenv("PATH") != orig {
		t.Fatal("PATH changed")
	}
}

func TestMiseStripShimsAndInstalls(t *testing.T) {
	data := t.TempDir()
	shim := filepath.Join(data, "mise", "shims")
	installBin := filepath.Join(data, "mise", "installs", "python", "latest", "bin")
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("MISE_SHIMS_DIR", shim)
	sys := filepath.Join(data, "usr", "bin")
	path := strings.Join([]string{shim, installBin, sys}, string(os.PathListSeparator))
	t.Setenv("PATH", path)

	w := mise.Wrap{LookPath: func(string) (string, error) { return "/usr/bin/mise", nil }}
	st := w.Begin(true)
	if !st.DidStrip {
		t.Fatal("want DidStrip")
	}
	if st.PathOrig != path {
		t.Fatalf("PathOrig = %q", st.PathOrig)
	}
	got := os.Getenv("PATH")
	if strings.Contains(got, shim) || strings.Contains(got, installBin) {
		t.Fatalf("PATH still has mise entries: %q", got)
	}
	if !strings.Contains(got, sys) {
		t.Fatalf("lost system PATH: %q", got)
	}
	env := st.ChildEnv([]string{"FOO=1", "PATH=" + path})
	joined := strings.Join(env, "\n")
	if strings.Contains(joined, "PATH="+path) {
		t.Fatalf("ChildEnv kept old PATH: %v", env)
	}
	if err := st.RestoreProcessPath(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("PATH") != path {
		t.Fatalf("restore PATH = %q want %q", os.Getenv("PATH"), path)
	}
}

func TestMiseRestoreNoop(t *testing.T) {
	orig := os.Getenv("PATH")
	st := mise.State{}
	if err := st.RestoreProcessPath(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("PATH") != orig {
		t.Fatal("PATH changed")
	}
}
