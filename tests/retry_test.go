package tests

import (
	"testing"

	"sysup-go/internal/program"
)

func TestAttemptsCap(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		gAlways bool
		gMax    int
		lAlways bool
		lMax    int
		want    int
	}{
		{name: "defaults", want: 1},
		{name: "recipe 10 no global cap", lMax: 10, want: 10},
		{name: "global 3 caps recipe 10", gMax: 3, lMax: 10, want: 3},
		{name: "global always 3", gAlways: true, gMax: 3, want: 3},
		{name: "recipe always uses global max", gMax: 3, lAlways: true, want: 3},
		{name: "recipe 1 under global 3", gMax: 3, lMax: 1, want: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := program.Attempts(tc.gAlways, tc.gMax, tc.lAlways, tc.lMax)
			if got != tc.want {
				t.Fatalf("Attempts = %d, want %d", got, tc.want)
			}
		})
	}
}
