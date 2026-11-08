package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpdateReportsUpToDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name": "v%s", "assets": []}`, Version)
	}))
	defer server.Close()

	original := latestReleaseURL
	latestReleaseURL = server.URL
	defer func() { latestReleaseURL = original }()

	out, err := runRoot(t, "update")
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if !strings.Contains(out, "Kon is up-to-date : v"+Version) {
		t.Fatalf("unexpected update output: %q", out)
	}
}

func TestReleaseAssetName(t *testing.T) {
	cases := map[string]string{
		"linux/amd64":   "kon_0.28.0-go_linux_amd64.tar.gz",
		"linux/arm64":   "kon_0.28.0-go_linux_arm64.tar.gz",
		"darwin/amd64":  "kon_0.28.0-go_darwin_amd64.tar.gz",
		"darwin/arm64":  "kon_0.28.0-go_darwin_arm64.tar.gz",
		"windows/amd64": "kon_0.28.0-go_windows_amd64.tar.gz",
	}

	for platform, want := range cases {
		got := releaseAssetName("v0.28.0-go", platform[:strings.Index(platform, "/")], platform[strings.Index(platform, "/")+1:])
		if got != want {
			t.Fatalf("releaseAssetName(%q) = %q, want %q", platform, got, want)
		}
	}
}
