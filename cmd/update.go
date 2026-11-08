package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const releaseIssueURL = "https://github.com/khanghua1505/kon/issues"

// latestReleaseURL is a variable so tests can point it at a stub server.
var latestReleaseURL = "https://api.github.com/repos/khanghua1505/kon/releases/latest"

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update kon to the latest release.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd)
		},
	}
}

func runUpdate(cmd *cobra.Command) error {
	client := &http.Client{Timeout: 30 * time.Second}

	release, err := fetchLatestRelease(client, latestReleaseURL)
	if err != nil {
		return err
	}

	if release.TagName == "v"+Version {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Kon is up-to-date : v%s\n", Version)
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "A new version of Kon is available (%s), the new version will be installed by replacing this binary.\n", release.TagName)

	url := ""
	for _, asset := range release.Assets {
		if asset.Name == releaseAssetName(release.TagName, runtime.GOOS, runtime.GOARCH) {
			url = asset.BrowserDownloadURL
			break
		}
	}

	if url == "" {
		return fmt.Errorf("Sorry, this release has no build for your OS, please create an issue : %s", releaseIssueURL)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Download url is: %s\n", url)

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	if err := replaceExecutable(executable, url, client); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Kon has been updated successfully: %s\n", executable)

	return nil
}

func fetchLatestRelease(client *http.Client, url string) (*githubRelease, error) {
	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no release published yet at %s, please create an issue: %s", latestReleaseURL, releaseIssueURL)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not fetch the latest release: %s", response.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// releaseAssetName builds the tar.gz asset name goreleaser publishes:
// kon_<version-without-v>_<goos>_<goarch>.tar.gz.
func releaseAssetName(tag, goos, goarch string) string {
	return fmt.Sprintf("kon_%s_%s_%s.tar.gz", strings.TrimPrefix(tag, "v"), goos, goarch)
}

// extractBinary reads a goreleaser tar.gz release archive and copies the kon
// binary member to dst.
func extractBinary(archive io.Reader, dst io.Writer) error {
	gz, err := gzip.NewReader(archive)
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return fmt.Errorf("no kon binary found in release archive")
		}
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeReg && filepath.Base(header.Name) == "kon" {
			_, err := io.Copy(dst, reader)
			return err
		}
	}
}

func replaceExecutable(path, url string, client *http.Client) error {
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("could not download %s: %s", url, response.Status)
	}

	temp, err := os.CreateTemp(filepath.Dir(path), ".kon-update-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer func() { _ = os.Remove(tempName) }()

	if err := extractBinary(response.Body, temp); err != nil {
		_ = temp.Close()
		return err
	}

	if err := temp.Close(); err != nil {
		return err
	}

	if err := os.Chmod(tempName, 0o755); err != nil {
		return err
	}

	return os.Rename(tempName, path)
}
