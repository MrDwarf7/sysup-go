package tests

import (
	"context"
	"errors"
	"testing"

	"sysup-go/internal/resolve"
)

func TestPkgManager(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name         string
		env          map[string]string
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
			env: map[string]string{
				resolve.EnvPkgManager: "",
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
			env: map[string]string{
				resolve.EnvPkgManager: "foo",
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
			env: map[string]string{
				resolve.EnvPkgManager: "foo",
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
			r := resolve.Resolver{
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
				if _, ok := errors.AsType[*resolve.Error](err); !ok {
					t.Fatalf("PkgManager() error %v is not *resolve.Error", err)
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
			if !tc.noParuYay {
				return
			}
			fallback := make(map[string]struct{}, len(resolve.FallbackHelpers))
			for _, h := range resolve.FallbackHelpers {
				fallback[h] = struct{}{}
			}
			for _, c := range calls {
				if _, ok := fallback[c]; ok {
					t.Fatalf("LookPath(%q) called, want no fallback probe", c)
				}
			}
		})
	}
}
