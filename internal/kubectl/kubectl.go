// Package kubectl wraps the kubectl binary used by kubie.
package kubectl

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/khanghua1505/kubie/internal/kubeconfig"
	"github.com/khanghua1505/kubie/internal/vars"
)

// Options controls one kubectl invocation.
type Options struct {
	// Binary is the kubectl executable. It defaults to "kubectl".
	Binary string
	// Context and Namespace are optional kubectl flags.
	Context   string
	Namespace string
	// KubeconfigPath selects an existing kubeconfig file.
	KubeconfigPath string
	// Config selects an isolated kubeconfig. When set, it is written to a
	// temporary file and removed after kubectl exits.
	Config *kubeconfig.KubeConfig
}

// GetNamespaces lists namespace names with kubectl.
func GetNamespaces(config *kubeconfig.KubeConfig) ([]string, error) {
	return GetNamespacesContext(context.Background(), Options{Config: config})
}

// GetNamespacesContext lists namespace names with kubectl and cancels the
// invocation when ctx is done.
func GetNamespacesContext(ctx context.Context, opts Options) ([]string, error) {
	binary := opts.Binary
	if binary == "" {
		binary = "kubectl"
	}

	kubeconfigPath := opts.KubeconfigPath
	removeConfig := false
	if opts.Config != nil {
		var err error
		kubeconfigPath, err = opts.Config.WriteTemp()
		if err != nil {
			return nil, fmt.Errorf("could not write temporary kubeconfig: %w", err)
		}
		removeConfig = true
		defer func() {
			if removeConfig {
				os.Remove(kubeconfigPath)
			}
		}()
	} else if kubeconfigPath == "" {
		var ok bool
		kubeconfigPath, ok = vars.KubeconfigPath()
		if !ok {
			return nil, fmt.Errorf("KUBIE_KUBECONFIG env variable missing")
		}
	}

	args := []string{"get", "namespaces"}
	if opts.Context != "" {
		args = append(args, "--context", opts.Context)
	}
	if opts.Namespace != "" {
		args = append(args, "--namespace", opts.Namespace)
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = kubeconfigEnv(kubeconfigPath)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Error calling kubectl:\n%s", stderr.String())
	}

	return parseNamespaces(stdout.String()), nil
}

func kubeconfigEnv(path string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "KUBECONFIG=") {
			continue
		}
		env = append(env, entry)
	}
	return append(env, "KUBECONFIG="+path)
}

func parseNamespaces(output string) []string {
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) == 0 {
		return nil
	}

	namespaces := make([]string, 0, len(lines)-1)
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		name := line
		if index := strings.IndexAny(name, " \t"); index >= 0 {
			name = name[:index]
		}
		if name != "" {
			namespaces = append(namespaces, name)
		}
	}
	return namespaces
}
