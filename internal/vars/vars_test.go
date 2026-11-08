package vars_test

import (
	"os"
	"testing"

	"github.com/khanghua1505/kon/internal/vars"
)

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("could not unset %s: %v", key, err)
	}
}

func TestDepth(t *testing.T) {
	tests := []struct {
		name  string
		value string
		set   bool
		want  int
	}{
		{name: "unset", want: 0},
		{name: "empty", value: "", set: true, want: 0},
		{name: "zero", value: "0", set: true, want: 0},
		{name: "one", value: "1", set: true, want: 1},
		{name: "nested", value: "3", set: true, want: 3},
		{name: "not a number", value: "abc", set: true, want: 0},
		{name: "negative", value: "-2", set: true, want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv(vars.EnvDepth, tc.value)
			} else {
				unsetEnv(t, vars.EnvDepth)
			}
			if got := vars.Depth(); got != tc.want {
				t.Fatalf("Depth() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestActive(t *testing.T) {
	tests := []struct {
		name  string
		value string
		set   bool
		want  bool
	}{
		{name: "unset", want: false},
		{name: "one", value: "1", set: true, want: true},
		{name: "zero", value: "0", set: true, want: false},
		{name: "empty", value: "", set: true, want: false},
		{name: "other value", value: "true", set: true, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv(vars.EnvActive, tc.value)
			} else {
				unsetEnv(t, vars.EnvActive)
			}
			if got := vars.Active(); got != tc.want {
				t.Fatalf("Active() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestEnsureActive(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		set     bool
		wantErr bool
	}{
		{name: "unset", wantErr: true},
		{name: "inactive", value: "0", set: true, wantErr: true},
		{name: "active", value: "1", set: true, wantErr: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv(vars.EnvActive, tc.value)
			} else {
				unsetEnv(t, vars.EnvActive)
			}

			err := vars.EnsureActive()
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("EnsureActive() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("EnsureActive() = nil, want error")
			}
			if err.Error() != "Not in kon shell!" {
				t.Fatalf("EnsureActive() error = %q, want %q", err.Error(), "Not in kon shell!")
			}
		})
	}
}

func TestPathAccessors(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		value string
		set   bool
	}{
		{name: "session unset", env: vars.EnvSession},
		{name: "session set", env: vars.EnvSession, value: "/tmp/kubie-session.json", set: true},
		{name: "kubeconfig unset", env: vars.EnvKubeconfig},
		{name: "kubeconfig set", env: vars.EnvKubeconfig, value: "/tmp/kubie-kubeconfig.yaml", set: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv(tc.env, tc.value)
			} else {
				unsetEnv(t, tc.env)
			}

			var got string
			var ok bool
			switch tc.env {
			case vars.EnvSession:
				got, ok = vars.SessionPath()
			case vars.EnvKubeconfig:
				got, ok = vars.KubeconfigPath()
			}

			if ok != tc.set {
				t.Fatalf("path lookup ok = %t, want %t", ok, tc.set)
			}
			if ok && got != tc.value {
				t.Fatalf("path = %q, want %q", got, tc.value)
			}
		})
	}
}
