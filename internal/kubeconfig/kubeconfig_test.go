package kubeconfig

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

var updateGolden = flag.Bool("update", false, "rewrite the golden files in testdata")

const sourceDirPlaceholder = "<SOURCE_DIR>"

func fixturePath(name string) string {
	return filepath.Join("testdata", "fixtures", name)
}

func readFixture(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(fixturePath(name))
	if err != nil {
		t.Fatalf("could not read fixture %s: %v", name, err)
	}

	return string(data)
}

func copyFixture(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(readFixture(t, name)), 0o600); err != nil {
		t.Fatalf("could not copy fixture %s: %v", name, err)
	}

	return path
}

func compareGolden(t *testing.T, name, got string) {
	t.Helper()

	if *updateGolden {
		if err := os.WriteFile(filepath.Join("testdata", name), []byte(got), 0o644); err != nil {
			t.Fatalf("could not write golden %s: %v", name, err)
		}
		return
	}

	want, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("could not read golden %s: %v", name, err)
	}

	if got != string(want) {
		t.Errorf("golden %s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, string(want))
	}
}

func contextNames(installed *Installed) []string {
	names := make([]string, 0, len(installed.Contexts))
	for _, context := range installed.Contexts {
		names = append(names, context.Item.Name)
	}

	return names
}

func TestLoadKubeconfigsMultiFile(t *testing.T) {
	installed := LoadKubeconfigs([]string{fixturePath("alpha.yaml"), fixturePath("beta.yaml")})

	if got := contextNames(installed); len(got) != 3 {
		t.Fatalf("expected 3 contexts, got %v", got)
	}
	if installed.Contexts[0].Item.Name != "alpha-dev" {
		t.Errorf("expected alpha-dev first, got %s", installed.Contexts[0].Item.Name)
	}
	if installed.Contexts[0].Source != fixturePath("alpha.yaml") {
		t.Errorf("expected the alpha source to be kept, got %s", installed.Contexts[0].Source)
	}
	if installed.Contexts[2].Source != fixturePath("beta.yaml") {
		t.Errorf("expected the beta source to be kept, got %s", installed.Contexts[2].Source)
	}

	if len(installed.Clusters) != 2 {
		t.Errorf("expected 2 clusters, got %d", len(installed.Clusters))
	}
	if len(installed.Users) != 2 {
		t.Errorf("expected 2 users, got %d", len(installed.Users))
	}

	cluster := installed.FindClusterByName("beta-cluster", fixturePath("beta.yaml"))
	if cluster == nil {
		t.Fatal("expected to find the beta cluster")
	}
	if cluster.Item.Cluster.Kind != yaml.MappingNode {
		t.Errorf("expected the cluster body to stay a raw mapping node, got kind %v", cluster.Item.Cluster.Kind)
	}
	if mappingValue(&cluster.Item.Cluster, "tls-server-name") == nil {
		t.Error("expected the unknown cluster field to be preserved")
	}

	kubeconfig, err := Load(fixturePath("beta.yaml"))
	if err != nil {
		t.Fatalf("could not load the beta kubeconfig: %v", err)
	}
	if _, ok := kubeconfig.Extra["x-vendor-extension"]; !ok {
		t.Error("expected the unknown top level key to be preserved")
	}
}

func TestGetInstalledContextsWithGlobs(t *testing.T) {
	glob := filepath.Join("testdata", "fixtures", "*.yaml")

	installed, err := GetInstalledContexts([]string{glob}, nil)
	if err != nil {
		t.Fatalf("could not load the installed contexts: %v", err)
	}
	if len(installed.Contexts) != 5 {
		t.Errorf("expected 5 contexts, got %v", contextNames(installed))
	}

	installed, err = GetInstalledContexts([]string{glob}, []string{fixturePath("beta.yaml")})
	if err != nil {
		t.Fatalf("could not load the installed contexts: %v", err)
	}
	if len(installed.Contexts) != 3 {
		t.Errorf("expected the excluded file to be dropped, got %v", contextNames(installed))
	}
	for _, context := range installed.Contexts {
		if context.Item.Name == "beta-dev" {
			t.Error("expected the excluded context to be dropped")
		}
	}
}

func TestGetInstalledContextsSkipsUnreadableFiles(t *testing.T) {
	installed, err := GetInstalledContexts([]string{fixturePath("alpha.yaml"), filepath.Join("testdata", "fixtures", "malformed", "*.yaml")}, nil)
	if err != nil {
		t.Fatalf("could not load the installed contexts: %v", err)
	}
	if len(installed.Contexts) != 1 || installed.Contexts[0].Item.Name != "alpha-dev" {
		t.Errorf("expected the malformed file to be skipped, got %v", contextNames(installed))
	}

	if _, err := GetInstalledContexts([]string{filepath.Join("testdata", "fixtures", "malformed", "*.yaml")}, nil); err == nil {
		t.Error("expected an error when no context could be found")
	} else if err.Error() != "Could not find any contexts in the Kubie kubeconfig directories!" {
		t.Errorf("unexpected error message: %s", err)
	}
}

func TestGetContextsMatching(t *testing.T) {
	installed := LoadKubeconfigs([]string{fixturePath("alpha.yaml"), fixturePath("beta.yaml")})

	matched := installed.GetContextsMatching("beta-*", false)
	if len(matched) != 2 {
		t.Fatalf("expected 2 matching contexts, got %d", len(matched))
	}
	if matched[0].Item.Name != "beta-dev" || matched[1].Item.Name != "beta-prod" {
		t.Errorf("expected the matches to be sorted by name, got %s and %s", matched[0].Item.Name, matched[1].Item.Name)
	}

	matched = installed.GetContextsMatching("alpha-* beta-*", true)
	if len(matched) != 3 {
		t.Fatalf("expected 3 matching contexts, got %d", len(matched))
	}
	if matched[0].Item.Name != "alpha-dev" {
		t.Errorf("expected alpha-dev first, got %s", matched[0].Item.Name)
	}

	matched = installed.GetContextsMatching("alpha-* beta-*", false)
	if len(matched) != 0 {
		t.Errorf("expected no match without multiple patterns, got %d", len(matched))
	}
}

func TestMakeKubeconfigForContextIsolatesContext(t *testing.T) {
	path := copyFixture(t, "beta.yaml")
	installed := LoadKubeconfigs([]string{path})

	kubeconfig, err := installed.MakeKubeconfigForContext("beta-dev", nil)
	if err != nil {
		t.Fatalf("could not isolate the context: %v", err)
	}

	if kubeconfig.CurrentContext != "beta-dev" {
		t.Errorf("expected current-context beta-dev, got %s", kubeconfig.CurrentContext)
	}
	if len(kubeconfig.Contexts) != 1 || len(kubeconfig.Clusters) != 1 || len(kubeconfig.Users) != 1 {
		t.Errorf("expected exactly one context, cluster and user, got %d, %d, %d", len(kubeconfig.Contexts), len(kubeconfig.Clusters), len(kubeconfig.Users))
	}

	if mappingValue(&kubeconfig.Users[0].User, "exec") == nil {
		t.Error("expected the exec credential plugin block to be preserved")
	}
	if mappingValue(&kubeconfig.Clusters[0].Cluster, "tls-server-name") == nil {
		t.Error("expected the unknown cluster field to be preserved")
	}

	certificate := mappingValue(&kubeconfig.Clusters[0].Cluster, "certificate-authority")
	if certificate == nil {
		t.Fatal("expected the certificate authority to be kept")
	}
	if !filepath.IsAbs(certificate.Value) {
		t.Errorf("expected an absolute certificate authority path, got %s", certificate.Value)
	}

	out, err := yaml.Marshal(kubeconfig)
	if err != nil {
		t.Fatalf("could not marshal the isolated kubeconfig: %v", err)
	}
	compareGolden(t, "isolate.golden", strings.ReplaceAll(string(out), filepath.Dir(path), sourceDirPlaceholder))
}

func TestMakeKubeconfigForContextSetsNamespace(t *testing.T) {
	path := copyFixture(t, "beta.yaml")
	installed := LoadKubeconfigs([]string{path})

	namespace := "prod"
	kubeconfig, err := installed.MakeKubeconfigForContext("beta-dev", &namespace)
	if err != nil {
		t.Fatalf("could not isolate the context: %v", err)
	}
	if kubeconfig.Contexts[0].Context.Namespace != "prod" {
		t.Errorf("expected the namespace to be set, got %s", kubeconfig.Contexts[0].Context.Namespace)
	}

	kubeconfig, err = installed.MakeKubeconfigForContext("beta-dev", nil)
	if err != nil {
		t.Fatalf("could not isolate the context: %v", err)
	}
	if kubeconfig.Contexts[0].Context.Namespace != "" {
		t.Errorf("expected the namespace to be dropped, got %s", kubeconfig.Contexts[0].Context.Namespace)
	}
}

func TestDeleteContextRewritesOriginalFile(t *testing.T) {
	path := copyFixture(t, "delete.yaml")
	other := copyFixture(t, "alpha.yaml")

	otherBefore, err := os.ReadFile(other)
	if err != nil {
		t.Fatalf("could not read the other kubeconfig: %v", err)
	}

	installed := LoadKubeconfigs([]string{path, other})
	if err := installed.DeleteContext("gone-ctx"); err != nil {
		t.Fatalf("could not delete the context: %v", err)
	}

	rewritten, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read the rewritten kubeconfig: %v", err)
	}
	compareGolden(t, "delete.golden", string(rewritten))

	otherAfter, err := os.ReadFile(other)
	if err != nil {
		t.Fatalf("could not read the other kubeconfig: %v", err)
	}
	if string(otherAfter) != string(otherBefore) {
		t.Error("expected the other kubeconfig to be left untouched")
	}

	remaining := LoadKubeconfigs([]string{path})
	if len(remaining.Contexts) != 1 || remaining.Contexts[0].Item.Name != "keep-ctx" {
		t.Errorf("expected keep-ctx to remain, got %v", contextNames(remaining))
	}

	rewrittenConfig, err := Load(path)
	if err != nil {
		t.Fatalf("could not load the rewritten kubeconfig: %v", err)
	}
	if _, ok := rewrittenConfig.Extra["x-vendor-extension"]; !ok {
		t.Error("expected the unknown top level key to survive the rewrite")
	}
	if rewrittenConfig.Preferences.Kind != yaml.MappingNode {
		t.Error("expected the preferences to survive the rewrite")
	}
	if remaining.FindClusterByName("keep-cluster", path) == nil {
		t.Error("expected the keep-cluster to remain")
	}
	if remaining.FindClusterByName("gone-cluster", path) != nil {
		t.Error("expected the gone-cluster to be removed")
	}
	if remaining.FindUserByName("gone-user", path) != nil {
		t.Error("expected the gone-user to be removed")
	}
}

func TestDeleteContextRemovesEmptyFile(t *testing.T) {
	path := copyFixture(t, "delete.yaml")

	installed := LoadKubeconfigs([]string{path})
	if err := installed.DeleteContext("gone-ctx"); err != nil {
		t.Fatalf("could not delete the context: %v", err)
	}

	installed = LoadKubeconfigs([]string{path})
	if err := installed.DeleteContext("keep-ctx"); err != nil {
		t.Fatalf("could not delete the context: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected the empty kubeconfig file to be removed, got %v", err)
	}
}

func TestLoadMalformedKubeconfig(t *testing.T) {
	if _, err := Load(filepath.Join("testdata", "fixtures", "malformed", "bad.yaml")); err == nil {
		t.Error("expected an error for a malformed kubeconfig")
	}
}

func TestWriteToFilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	kubeconfig := &KubeConfig{APIVersion: "v1", Kind: "Config"}
	if err := kubeconfig.WriteToFile(path); err != nil {
		t.Fatalf("could not write the kubeconfig: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("could not stat the kubeconfig: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected 0600 permissions, got %v", info.Mode().Perm())
	}

	temp, err := kubeconfig.WriteTemp()
	if err != nil {
		t.Fatalf("could not write the temporary kubeconfig: %v", err)
	}
	defer os.Remove(temp)

	if !strings.HasPrefix(filepath.Base(temp), "kon-config") || !strings.HasSuffix(temp, ".yaml") {
		t.Errorf("expected a kon-config*.yaml temporary file, got %s", temp)
	}

	info, err = os.Stat(temp)
	if err != nil {
		t.Fatalf("could not stat the temporary kubeconfig: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected 0600 permissions, got %v", info.Mode().Perm())
	}
}

func TestManualQAIsolate(t *testing.T) {
	path := copyFixture(t, "beta.yaml")
	installed := LoadKubeconfigs([]string{path})

	kubeconfig, err := installed.MakeKubeconfigForContext("beta-dev", nil)
	if err != nil {
		t.Fatalf("could not isolate the context: %v", err)
	}

	temp, err := kubeconfig.WriteTemp()
	if err != nil {
		t.Fatalf("could not write the temporary kubeconfig: %v", err)
	}
	defer os.Remove(temp)

	content, err := os.ReadFile(temp)
	if err != nil {
		t.Fatalf("could not read the temporary kubeconfig: %v", err)
	}

	t.Logf("isolated kubeconfig written to %s", temp)
	t.Logf("isolated kubeconfig content:\n%s", string(content))
}
