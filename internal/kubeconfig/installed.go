package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Installed holds every cluster, context and user found in the loaded kubeconfig files.
type Installed struct {
	Clusters []Sourced[NamedCluster]
	Contexts []Sourced[NamedContext]
	Users    []Sourced[NamedUser]
}

// ResolvePaths expands the include globs and drops the paths matched by the exclude globs.
func ResolvePaths(includeGlobs, excludeGlobs []string) ([]string, error) {
	paths := map[string]struct{}{}

	for _, pattern := range includeGlobs {
		matches, err := filepath.Glob(expandUser(pattern))
		if err != nil {
			return nil, fmt.Errorf("invalid glob %q: %w", pattern, err)
		}
		for _, match := range matches {
			paths[match] = struct{}{}
		}
	}

	for _, pattern := range excludeGlobs {
		matches, err := filepath.Glob(expandUser(pattern))
		if err != nil {
			return nil, fmt.Errorf("invalid glob %q: %w", pattern, err)
		}
		for _, match := range matches {
			delete(paths, match)
		}
	}

	resolved := make([]string, 0, len(paths))
	for path := range paths {
		resolved = append(resolved, path)
	}
	sort.Strings(resolved)

	return resolved, nil
}

// LoadKubeconfigs loads every given kubeconfig file. Files that cannot be read
// or parsed are skipped with a warning.
func LoadKubeconfigs(paths []string) *Installed {
	installed := &Installed{}

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}

		kubeconfig, err := Load(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading kubeconfig %s: %v\n", path, err)
			continue
		}

		for _, cluster := range kubeconfig.Clusters {
			installed.Clusters = append(installed.Clusters, Sourced[NamedCluster]{Source: path, Item: cluster})
		}
		for _, context := range kubeconfig.Contexts {
			installed.Contexts = append(installed.Contexts, Sourced[NamedContext]{Source: path, Item: context})
		}
		for _, user := range kubeconfig.Users {
			installed.Users = append(installed.Users, Sourced[NamedUser]{Source: path, Item: user})
		}
	}

	return installed
}

// GetInstalledContexts loads every kubeconfig matched by the include globs,
// minus the ones matched by the exclude globs.
func GetInstalledContexts(includeGlobs, excludeGlobs []string) (*Installed, error) {
	paths, err := ResolvePaths(includeGlobs, excludeGlobs)
	if err != nil {
		return nil, err
	}

	installed := LoadKubeconfigs(paths)
	if len(installed.Contexts) == 0 {
		return nil, fmt.Errorf("Could not find any contexts in the Kubie kubeconfig directories!")
	}

	return installed, nil
}

// GetKubeconfigsContexts loads the given kubeconfig files.
func GetKubeconfigsContexts(paths []string) (*Installed, error) {
	installed := LoadKubeconfigs(paths)
	if len(installed.Contexts) == 0 {
		return nil, fmt.Errorf("Could not find any contexts in the given set of files!")
	}

	return installed, nil
}

// FindContextByName returns the context with the given name, if any.
func (i *Installed) FindContextByName(name string) *Sourced[NamedContext] {
	for idx := range i.Contexts {
		if i.Contexts[idx].Item.Name == name {
			return &i.Contexts[idx]
		}
	}

	return nil
}

// FindClusterByName returns the cluster with the given name, preferring the one
// defined in source and falling back to any cluster with that name.
func (i *Installed) FindClusterByName(name, source string) *Sourced[NamedCluster] {
	for idx := range i.Clusters {
		if i.Clusters[idx].Item.Name == name && i.Clusters[idx].Source == source {
			return &i.Clusters[idx]
		}
	}

	for idx := range i.Clusters {
		if i.Clusters[idx].Item.Name == name {
			return &i.Clusters[idx]
		}
	}

	return nil
}

// FindUserByName returns the user with the given name, preferring the one
// defined in source and falling back to any user with that name.
func (i *Installed) FindUserByName(name, source string) *Sourced[NamedUser] {
	for idx := range i.Users {
		if i.Users[idx].Item.Name == name && i.Users[idx].Source == source {
			return &i.Users[idx]
		}
	}

	for idx := range i.Users {
		if i.Users[idx].Item.Name == name {
			return &i.Users[idx]
		}
	}

	return nil
}

// FindContextsByCluster returns the contexts referencing the given cluster in source.
func (i *Installed) FindContextsByCluster(name, source string) []*Sourced[NamedContext] {
	var found []*Sourced[NamedContext]

	for idx := range i.Contexts {
		if i.Contexts[idx].Item.Context.Cluster == name && i.Contexts[idx].Source == source {
			found = append(found, &i.Contexts[idx])
		}
	}

	return found
}

// FindContextsByUser returns the contexts referencing the given user in source.
func (i *Installed) FindContextsByUser(name, source string) []*Sourced[NamedContext] {
	var found []*Sourced[NamedContext]

	for idx := range i.Contexts {
		if i.Contexts[idx].Item.Context.User == name && i.Contexts[idx].Source == source {
			found = append(found, &i.Contexts[idx])
		}
	}

	return found
}

// GetContextsMatching returns the contexts whose name matches the pattern.
// When allowMultiple is set, the pattern is a whitespace separated list of
// patterns.
func (i *Installed) GetContextsMatching(pattern string, allowMultiple bool) []*Sourced[NamedContext] {
	patterns := []string{pattern}
	if allowMultiple {
		patterns = strings.Fields(pattern)
	}

	var matched []*Sourced[NamedContext]
	for _, current := range patterns {
		var found []*Sourced[NamedContext]

		for idx := range i.Contexts {
			ok, err := filepath.Match(current, i.Contexts[idx].Item.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARNING: invalid context pattern %q: %v\n", current, err)
				continue
			}
			if ok {
				found = append(found, &i.Contexts[idx])
			}
		}

		if len(patterns) > 1 && len(found) == 0 {
			fmt.Printf("WARNING: No context matching %s\n", current)
		}

		matched = append(matched, found...)
	}

	sort.SliceStable(matched, func(a, b int) bool {
		return matched[a].Item.Name < matched[b].Item.Name
	})

	deduped := matched[:0]
	for idx, context := range matched {
		if idx > 0 && context.Item.Name == matched[idx-1].Item.Name {
			continue
		}
		deduped = append(deduped, context)
	}

	return deduped
}

// DeleteContext removes the context from the file it was loaded from, along
// with the cluster and user it referenced. The file is removed when nothing is
// left in it.
func (i *Installed) DeleteContext(name string) error {
	context := i.FindContextByName(name)
	if context == nil {
		return fmt.Errorf("Context not found")
	}

	kubeconfig, err := Load(context.Source)
	if err != nil {
		return err
	}

	clusterName := context.Item.Context.Cluster
	userName := context.Item.Context.User

	var contexts []NamedContext
	for _, item := range kubeconfig.Contexts {
		if item.Name != name {
			contexts = append(contexts, item)
		}
	}

	var clusters []NamedCluster
	for _, item := range kubeconfig.Clusters {
		if item.Name != clusterName {
			clusters = append(clusters, item)
		}
	}

	var users []NamedUser
	for _, item := range kubeconfig.Users {
		if item.Name != userName {
			users = append(users, item)
		}
	}

	kubeconfig.Contexts = contexts
	kubeconfig.Clusters = clusters
	kubeconfig.Users = users

	if len(contexts) == 0 && len(clusters) == 0 && len(users) == 0 {
		fmt.Printf("Deleting kubeconfig %s because is it now empty.\n", context.Source)
		if err := os.Remove(context.Source); err != nil {
			return fmt.Errorf("Could not delete empty kubeconfig file: %w", err)
		}

		return nil
	}

	fmt.Printf("Updating kubeconfig %s.\n", context.Source)
	if err := kubeconfig.WriteToFile(context.Source); err != nil {
		return fmt.Errorf("Could not open kubeconfig file to rewrite it: %w", err)
	}

	return nil
}

// MakeKubeconfigForContext builds a kubeconfig holding only the given context,
// its cluster and its user. Relative certificate paths are resolved against
// the directory of the file the context came from.
func (i *Installed) MakeKubeconfigForContext(contextName string, namespace *string) (*KubeConfig, error) {
	context := i.FindContextByName(contextName)
	if context == nil {
		return nil, fmt.Errorf("Could not find context %s", contextName)
	}

	isolated := context.Item
	isolated.Context.Namespace = ""
	if namespace != nil {
		isolated.Context.Namespace = *namespace
	}

	sourceDir, err := filepath.Abs(filepath.Dir(context.Source))
	if err != nil {
		return nil, err
	}

	clusterSource := i.FindClusterByName(isolated.Context.Cluster, context.Source)
	if clusterSource == nil {
		return nil, fmt.Errorf("Could not find cluster %s referenced by context %s", isolated.Context.Cluster, contextName)
	}

	cluster := clusterSource.Item
	cluster.Cluster = *cloneNode(&clusterSource.Item.Cluster)
	makePathAbsolute(&cluster.Cluster, "certificate-authority", sourceDir)

	userSource := i.FindUserByName(isolated.Context.User, context.Source)
	if userSource == nil {
		return nil, fmt.Errorf("Could not find user %s referenced by context %s", isolated.Context.User, contextName)
	}

	user := userSource.Item
	user.User = *cloneNode(&userSource.Item.User)
	makePathAbsolute(&user.User, "client-certificate", sourceDir)
	makePathAbsolute(&user.User, "client-key", sourceDir)

	return &KubeConfig{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: contextName,
		Clusters:       []NamedCluster{cluster},
		Contexts:       []NamedContext{isolated},
		Users:          []NamedUser{user},
	}, nil
}

// expandUser replaces a leading ~ with the home directory.
func expandUser(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}
