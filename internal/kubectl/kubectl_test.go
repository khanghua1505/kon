package kubectl

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/khanghua1505/kon/internal/kubeconfig"
)

const namespaceTable = "NAME              STATUS   AGE\n" +
	"default           Active   1d\n" +
	"kube-system       Active   1d\n"

func writeKubectlShim(t *testing.T, dir, table string) string {
	t.Helper()

	path := filepath.Join(dir, "kubectl")
	script := "#!/bin/sh\n" +
		"if [ -n \"$KON_TEST_CAPTURE\" ]; then\n" +
		"  { printf 'KUBECONFIG=%s\\n' \"$KUBECONFIG\"; printf 'KON_KUBECONFIG=%s\\n' \"$KON_KUBECONFIG\"; } > \"$KON_TEST_CAPTURE\"\n" +
		"fi\n" +
		"cat <<'TABLE'\n" + table + "TABLE\n"

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}

	return path
}

func installShim(t *testing.T, dir string) {
	t.Helper()

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestGetNamespacesContextParsesNamespaceTable(t *testing.T) {
	dir := t.TempDir()
	writeKubectlShim(t, dir, namespaceTable)
	installShim(t, dir)
	t.Setenv("KON_KUBECONFIG", filepath.Join(dir, "kubeconfig.yaml"))

	namespaces, err := GetNamespacesContext(context.Background(), Options{})
	if err != nil {
		t.Fatalf("get namespaces failed: %v", err)
	}

	want := []string{"default", "kube-system"}
	if !reflect.DeepEqual(namespaces, want) {
		t.Fatalf("unexpected namespaces: got %v, want %v", namespaces, want)
	}
}

func TestGetNamespacesContextKeepsRowsAfterHeader(t *testing.T) {
	dir := t.TempDir()
	writeKubectlShim(t, dir, "NAME\n----\ndefault\nkube-system\n")
	installShim(t, dir)
	t.Setenv("KON_KUBECONFIG", filepath.Join(dir, "kubeconfig.yaml"))

	namespaces, err := GetNamespacesContext(context.Background(), Options{})
	if err != nil {
		t.Fatalf("get namespaces failed: %v", err)
	}

	want := []string{"----", "default", "kube-system"}
	if !reflect.DeepEqual(namespaces, want) {
		t.Fatalf("unexpected namespaces: got %v, want %v", namespaces, want)
	}
}

func TestGetNamespacesContextUsesKubieKubeconfigEnv(t *testing.T) {
	dir := t.TempDir()
	writeKubectlShim(t, dir, namespaceTable)
	installShim(t, dir)

	kubeconfigPath := filepath.Join(dir, "kubeconfig.yaml")
	if err := os.WriteFile(kubeconfigPath, []byte("apiVersion: v1\nkind: Config\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("KON_KUBECONFIG", kubeconfigPath)

	capture := filepath.Join(dir, "capture.txt")
	t.Setenv("KON_TEST_CAPTURE", capture)

	if _, err := GetNamespacesContext(context.Background(), Options{}); err != nil {
		t.Fatalf("get namespaces failed: %v", err)
	}

	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatalf("kubectl shim did not record the environment: %v", err)
	}

	want := "KUBECONFIG=" + kubeconfigPath + "\nKON_KUBECONFIG=" + kubeconfigPath + "\n"
	if string(data) != want {
		t.Fatalf("unexpected kubectl environment\n--- got ---\n%s--- want ---\n%s", data, want)
	}
}

func TestGetNamespacesContextRemovesTemporaryKubeconfig(t *testing.T) {
	dir := t.TempDir()
	writeKubectlShim(t, dir, namespaceTable)
	installShim(t, dir)

	envPath := filepath.Join(dir, "kubeconfig.yaml")
	t.Setenv("KON_KUBECONFIG", envPath)

	capture := filepath.Join(dir, "capture.txt")
	t.Setenv("KON_TEST_CAPTURE", capture)

	config := &kubeconfig.KubeConfig{APIVersion: "v1", Kind: "Config"}
	if _, err := GetNamespacesContext(context.Background(), Options{Config: config}); err != nil {
		t.Fatalf("get namespaces failed: %v", err)
	}

	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatalf("kubectl shim did not record the environment: %v", err)
	}

	first := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)[0]
	tempPath := strings.TrimPrefix(first, "KUBECONFIG=")
	if tempPath == "" || tempPath == envPath {
		t.Fatalf("expected a temporary kubeconfig, got %q", tempPath)
	}

	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("temporary kubeconfig %s was not removed", tempPath)
	}
}
