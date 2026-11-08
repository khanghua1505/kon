// Package kubeconfig loads, isolates and prunes kubeconfig documents.
//
// Every item keeps the file it was loaded from, so commands can rewrite the
// original file instead of a merged view. Cluster and user bodies are kept as
// raw YAML nodes: unknown fields and exec credential plugin blocks must
// survive a load/rewrite round trip.
package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Sourced couples an item with the kubeconfig file it was loaded from.
type Sourced[T any] struct {
	Source string
	Item   T
}

// NamedCluster is an entry of the `clusters` list.
type NamedCluster struct {
	Name    string    `yaml:"name"`
	Cluster yaml.Node `yaml:"cluster"`
}

// NamedUser is an entry of the `users` list.
type NamedUser struct {
	Name string    `yaml:"name"`
	User yaml.Node `yaml:"user"`
}

// Context is the body of a context entry.
type Context struct {
	Cluster   string               `yaml:"cluster"`
	Namespace string               `yaml:"namespace,omitempty"`
	User      string               `yaml:"user"`
	Extra     map[string]yaml.Node `yaml:",inline"`
}

// NamedContext is an entry of the `contexts` list.
type NamedContext struct {
	Name    string  `yaml:"name"`
	Context Context `yaml:"context"`
}

// KubeConfig mirrors the kubeconfig document. Unknown top level keys are kept
// in Extra instead of being dropped.
type KubeConfig struct {
	APIVersion     string               `yaml:"apiVersion,omitempty"`
	Kind           string               `yaml:"kind,omitempty"`
	CurrentContext string               `yaml:"current-context,omitempty"`
	Preferences    yaml.Node            `yaml:"preferences,omitempty"`
	Clusters       []NamedCluster       `yaml:"clusters,omitempty"`
	Contexts       []NamedContext       `yaml:"contexts,omitempty"`
	Users          []NamedUser          `yaml:"users,omitempty"`
	Extra          map[string]yaml.Node `yaml:",inline"`
}

// Load reads and parses a kubeconfig file.
func Load(path string) (*KubeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var kubeconfig KubeConfig
	if err := yaml.Unmarshal(data, &kubeconfig); err != nil {
		return nil, fmt.Errorf("could not parse kubeconfig %s: %w", path, err)
	}

	return &kubeconfig, nil
}

// WriteToFile writes the kubeconfig to path, forcing 0600 permissions.
func (kc *KubeConfig) WriteToFile(path string) error {
	data, err := yaml.Marshal(kc)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}

	return os.Chmod(path, 0o600)
}

// WriteTemp writes the kubeconfig to a new temporary file named
// kubie-config*.yaml with 0600 permissions and returns its path.
func (kc *KubeConfig) WriteTemp() (string, error) {
	file, err := os.CreateTemp("", "kubie-config*.yaml")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}

	if err := kc.WriteToFile(path); err != nil {
		_ = os.Remove(path)
		return "", err
	}

	return path, nil
}

// cloneNode deep copies a YAML node so mutations never leak into the
// document the node was loaded from.
func cloneNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}

	clone := *node
	clone.Content = make([]*yaml.Node, len(node.Content))
	for i, child := range node.Content {
		clone.Content[i] = cloneNode(child)
	}
	if node.Alias != nil {
		clone.Alias = cloneNode(node.Alias)
	}

	return &clone
}

// mappingValue returns the value node of key in a mapping node.
func mappingValue(body *yaml.Node, key string) *yaml.Node {
	if body == nil || body.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(body.Content); i += 2 {
		if body.Content[i].Value == key {
			return body.Content[i+1]
		}
	}

	return nil
}

// makePathAbsolute rewrites a relative path value into an absolute one,
// resolved against dir.
func makePathAbsolute(body *yaml.Node, key, dir string) {
	value := mappingValue(body, key)
	if value == nil || value.Kind != yaml.ScalarNode || value.Value == "" {
		return
	}

	if filepath.IsAbs(value.Value) {
		return
	}

	value.SetString(filepath.Join(dir, value.Value))
}
