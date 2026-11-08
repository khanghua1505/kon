// Package vars reads the KON_* environment variables that describe the kon
// shell a command is running in.
//
// KON_* variables take precedence; the legacy KUBIE_* variables are still
// read as a fallback so shells spawned by the original kubie keep working.
package vars

import (
	"errors"
	"os"
	"strconv"
)

// Legacy kubie names stay readable for backward compatibility.
const (
	EnvDepth      = "KON_DEPTH"
	EnvActive     = "KON_ACTIVE"
	EnvSession    = "KON_SESSION"
	EnvKubeconfig = "KON_KUBECONFIG"

	EnvAltKubeconfig   = "KON_KUBE_CONFIG"
	EnvKubieDepth      = "KUBIE_DEPTH"
	EnvKubieActive     = "KUBIE_ACTIVE"
	EnvKubieSession    = "KUBIE_SESSION"
	EnvKubieKubeconfig = "KUBIE_KUBECONFIG"
)

// ErrNotInKonShell is returned by EnsureActive when the command does not run
// inside a kon shell.
var ErrNotInKonShell = errors.New("Not in kon shell!")

// Depth returns the current shell depth. It is 0 outside a kon shell and
// when neither KON_DEPTH nor KUBIE_DEPTH is set to a valid depth.
func Depth() int {
	value, _ := GetEnv(EnvDepth, EnvKubieDepth)
	depth, err := strconv.Atoi(value)
	if err != nil || depth < 0 {
		return 0
	}
	return depth
}

// Active reports whether the command runs inside a kon shell.
func Active() bool {
	value, _ := GetEnv(EnvActive, EnvKubieActive)
	return value == "1"
}

// EnsureActive returns ErrNotInKonShell when the command does not run inside
// a kon shell.
func EnsureActive() error {
	if !Active() {
		return ErrNotInKonShell
	}
	return nil
}

// SessionPath returns the path of the session file and whether it is set.
func SessionPath() (string, bool) {
	return GetEnv(EnvSession, EnvKubieSession)
}

// KubeconfigPath returns the path of the temporary kubeconfig and whether it
// is set. Both KON_KUBECONFIG and KON_KUBE_CONFIG spellings are accepted.
func KubeconfigPath() (string, bool) {
	return GetEnv(EnvKubeconfig, EnvAltKubeconfig, EnvKubieKubeconfig)
}

// GetEnv returns the value of the first variable of name, fallback... that is
// set to a non-empty value. ok reports whether any variable was set.
func GetEnv(name string, fallback ...string) (string, bool) {
	for _, n := range append([]string{name}, fallback...) {
		if value, ok := os.LookupEnv(n); ok && value != "" {
			return value, true
		}
	}
	return "", false
}
