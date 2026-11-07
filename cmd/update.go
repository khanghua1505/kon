package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

const releaseIssueURL = "https://github.com/khanghua1505/kubie/issues"

// latestReleaseURL is a variable so tests can point it at a stub server.
var latestReleaseURL = "https://api.github.com/repos/khanghua1505/kubie/releases/latest"

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
		Short: "Update kubie to the latest release.",
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
		fmt.Fprintf(cmd.OutOrStdout(), "Kubie is up-to-date : v%s\n", Version)
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "A new version of Kubie is available (%s), the new version will be installed by replacing this binary.\n", release.TagName)

	url := ""
	for _, asset := range release.Assets {
		if asset.Name == releaseBinaryName(runtime.GOOS, runtime.GOARCH) {
			url = asset.BrowserDownloadURL
			break
		}
	}

	if url == "" {
		return fmt.Errorf("Sorry, this release has no build for your OS, please create an issue : %s", releaseIssueURL)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Download url is: %s\n", url)

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	if err := replaceExecutable(executable, url, client); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Kubie has been updated successfully: %s\n", executable)

	return nil
}

func fetchLatestRelease(client *http.Client, url string) (*githubRelease, error) {
	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

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

func releaseBinaryName(goos, goarch string) string {
	switch goos + "/" + goarch {
	case "linux/amd64":
		return "kubie-linux-amd64"
	case "linux/arm":
		return "kubie-linux-arm32"
	case "linux/arm64":
		return "kubie-linux-arm64"
	case "darwin/amd64":
		return "kubie-darwin-amd64"
	case "darwin/arm64":
		return "kubie-darwin-arm64"
	default:
		return "kubie-" + goos + "-" + goarch
	}
}

func replaceExecutable(path, url string, client *http.Client) error {
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("could not download %s: %s", url, response.Status)
	}

	temp, err := os.CreateTemp(filepath.Dir(path), ".kubie-update-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if _, err := io.Copy(temp, response.Body); err != nil {
		temp.Close()
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
