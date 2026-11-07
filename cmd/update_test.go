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

	if !strings.Contains(out, "Kubie is up-to-date : v"+Version) {
		t.Fatalf("unexpected update output: %q", out)
	}
}

func TestReleaseBinaryName(t *testing.T) {
	cases := map[string]string{
		"linux/amd64":   "kubie-linux-amd64",
		"linux/arm":     "kubie-linux-arm32",
		"linux/arm64":   "kubie-linux-arm64",
		"darwin/amd64":  "kubie-darwin-amd64",
		"darwin/arm64":  "kubie-darwin-arm64",
		"windows/amd64": "kubie-windows-amd64",
	}

	for platform, want := range cases {
		got := releaseBinaryName(platform[:strings.Index(platform, "/")], platform[strings.Index(platform, "/")+1:])
		if got != want {
			t.Fatalf("releaseBinaryName(%q) = %q, want %q", platform, got, want)
		}
	}
}
