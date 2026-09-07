package resolve

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestPkgManager(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name         string
		env          map[string]*string
		paths        map[string]string
		cfgName      string
		ctx          context.Context
		want         string
		wantErr      bool
		wantCanceled bool
		noLookPath   bool
		noParuYay    bool
	}{
		{
			name: "unset env, paru and yay both found",
			paths: map[string]string{
				"paru": "/usr/bin/paru",
				"yay":  "/usr/bin/yay",
			},
			want: "/usr/bin/paru",
		},
		{
			name: "unset env, only yay",
			paths: map[string]string{
				"yay": "/usr/bin/yay",
			},
			want: "/usr/bin/yay",
		},
		{
			name: "unset env, neither, cfgName found",
			paths: map[string]string{
				"custom": "/opt/custom",
			},
			cfgName: "custom",
			want:    "/opt/custom",
		},
		{
			name:    "unset env, neither, empty cfg",
			wantErr: true,
		},
		{
			name: "empty PKG_MANAGER",
			env: map[string]*string{
				"PKG_MANAGER": ptr(""),
			},
			paths: map[string]string{
				"paru": "/usr/bin/paru",
				"yay":  "/usr/bin/yay",
			},
			wantErr:    true,
			noLookPath: true,
		},
		{
			name: "PKG_MANAGER set, missing",
			env: map[string]*string{
				"PKG_MANAGER": ptr("foo"),
			},
			paths: map[string]string{
				"paru": "/usr/bin/paru",
				"yay":  "/usr/bin/yay",
			},
			wantErr:   true,
			noParuYay: true,
		},
		{
			name: "PKG_MANAGER set, found",
			env: map[string]*string{
				"PKG_MANAGER": ptr("foo"),
			},
			paths: map[string]string{
				"foo":  "/bin/foo",
				"paru": "/usr/bin/paru",
				"yay":  "/usr/bin/yay",
			},
			want:      "/bin/foo",
			noParuYay: true,
		},
		{
			name:         "cancelled context",
			ctx:          canceled,
			paths:        map[string]string{"paru": "/usr/bin/paru"},
			cfgName:      "custom",
			wantErr:      true,
			wantCanceled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeEnvPath{
				env:   tc.env,
				paths: tc.paths,
			}
			r := Resolver{
				LookPath:  f.lookPath,
				LookupEnv: f.lookupEnv,
			}
			ctx := tc.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := r.PkgManager(ctx, tc.cfgName)
			if tc.wantErr {
				if err == nil {
					t.Fatal("PkgManager() error = nil, want error")
				}
				if tc.wantCanceled && !errors.Is(err, context.Canceled) {
					t.Fatalf("PkgManager() error = %v, want context.Canceled", err)
				}
				var re *Error
				if !errors.As(err, &re) {
					t.Fatalf("PkgManager() error %v is not *Error", err)
				}
			} else if err != nil {
				t.Fatalf("PkgManager() unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("PkgManager() = %q, want %q", got, tc.want)
			}

			calls := f.called()
			if tc.noLookPath && len(calls) != 0 {
				t.Fatalf("LookPath called %v, want 0 calls", calls)
			}
			if tc.noParuYay {
				for _, c := range calls {
					if c == "paru" || c == "yay" {
						t.Fatalf("LookPath(%q) called, want no paru/yay probe", c)
					}
				}
			}
		})
	}
}

func ptr(s string) *string { return &s }

// fakeEnvPath injects PATH and env. env uses a three-state map: missing
// key is unset, pointer to "" is set-but-empty, pointer to a value is set.
type fakeEnvPath struct {
	env   map[string]*string
	paths map[string]string

	mu    sync.Mutex
	calls []string
}

func (f *fakeEnvPath) lookPath(file string) (string, error) {
	f.mu.Lock()
	f.calls = append(f.calls, file)
	f.mu.Unlock()
	if p, ok := f.paths[file]; ok {
		return p, nil
	}
	return "", errors.New("not found")
}

func (f *fakeEnvPath) lookupEnv(key string) (string, bool) {
	v, ok := f.env[key]
	if !ok || v == nil {
		return "", false
	}
	return *v, true
}

func (f *fakeEnvPath) called() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}
