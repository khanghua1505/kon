// Package vars reads the KUBIE_* environment variables that describe the
// kubie shell a command is running in.
//
// The environment variable names are byte-identical to the Rust implementation
// in /tmp/kubie-src/src/vars.rs.
package vars

import (
	"errors"
	"os"
	"strconv"
)

// Environment variable names used by kubie.
const (
	EnvDepth      = "KUBIE_DEPTH"
	EnvActive     = "KUBIE_ACTIVE"
	EnvSession    = "KUBIE_SESSION"
	EnvKubeconfig = "KUBIE_KUBECONFIG"
)

// ErrNotInKubieShell is returned by EnsureActive when the command does not run
// inside a kubie shell.
var ErrNotInKubieShell = errors.New("Not in kubie shell!")

// Depth returns the current shell depth. It is 0 outside a kubie shell and
// when KUBIE_DEPTH is missing or not a valid depth.
func Depth() int {
	depth, err := strconv.Atoi(os.Getenv(EnvDepth))
	if err != nil || depth < 0 {
		return 0
	}
	return depth
}

// Active reports whether the command runs inside a kubie shell.
func Active() bool {
	return os.Getenv(EnvActive) == "1"
}

// EnsureActive returns ErrNotInKubieShell when the command does not run inside
// a kubie shell.
func EnsureActive() error {
	if !Active() {
		return ErrNotInKubieShell
	}
	return nil
}

// SessionPath returns the path of the session file and whether it is set.
func SessionPath() (string, bool) {
	return lookup(EnvSession)
}

// KubeconfigPath returns the path of the temporary kubeconfig and whether it
// is set.
func KubeconfigPath() (string, bool) {
	return lookup(EnvKubeconfig)
}

func lookup(name string) (string, bool) {
	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		return "", false
	}
	return value, true
}
