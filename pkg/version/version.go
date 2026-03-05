package version

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xivstrings/pkg/store"
)

const (
	githubReleasesURL = "https://api.github.com/repos/thewakingsands/ixion/releases/latest"
	stringsZipName    = "strings.zip"
)

// ReleaseAsset represents a GitHub release asset.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// GitHubRelease represents the latest release API response.
type GitHubRelease struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
}

// GetLatestRelease fetches the latest release from GitHub.
func GetLatestRelease() (*GitHubRelease, error) {
	req, err := http.NewRequest(http.MethodGet, githubReleasesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github api status %d: %s", resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &release, nil
}

// VersionFromTag normalizes tag_name for use as directory name (e.g. "publish/20260303-8b409c8" -> "publish-20260303-8b409c8").
func VersionFromTag(tag string) string {
	return strings.ReplaceAll(strings.TrimSpace(tag), "/", "-")
}

// GetStringsZipURL returns the download URL for strings.zip from the release, or empty string if not found.
func GetStringsZipURL(r *GitHubRelease) string {
	for _, a := range r.Assets {
		if a.Name == stringsZipName {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

// GetLocalVersion reads the current version from baseDir/version file.
func GetLocalVersion(baseDir string) (string, error) {
	p := filepath.Join(baseDir, "version")
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read version file: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

// WriteVersion writes the version string to baseDir/version.
func WriteVersion(baseDir string, version string) error {
	p := filepath.Join(baseDir, "version")
	return os.WriteFile(p, []byte(version+"\n"), 0644)
}

// UpdateToVersion downloads strings.zip from zipURL, extracts to data/[version], builds index to index/[version], and writes version file.
func UpdateToVersion(baseDir string, version string, zipURL string) error {
	stringDir := filepath.Join(baseDir, "strings", version)
	indexDir := filepath.Join(baseDir, "index", version)

	if err := os.MkdirAll(stringDir, 0755); err != nil {
		return fmt.Errorf("mkdir data dir: %w", err)
	}

	tmpZip := filepath.Join(os.TempDir(), "xivstrings-strings.zip")
	if err := downloadFile(zipURL, tmpZip); err != nil {
		return fmt.Errorf("download strings.zip: %w", err)
	}
	defer os.Remove(tmpZip)

	if err := extractZip(tmpZip, stringDir); err != nil {
		return fmt.Errorf("extract zip: %w", err)
	}

	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return fmt.Errorf("mkdir index dir: %w", err)
	}
	if err := store.BuildIndex(stringDir, indexDir); err != nil {
		return fmt.Errorf("build index: %w", err)
	}

	if err := WriteVersion(baseDir, version); err != nil {
		return fmt.Errorf("write version: %w", err)
	}
	log.Printf("Updated to version %s (data: %s, index: %s)", version, stringDir, indexDir)
	return nil
}

// EnsureResult holds the current version and paths after ensuring data is present.
type EnsureResult struct {
	Version   string // current version (tag normalized for filesystem)
	StringDir string // absolute path to strings/[version]
	IndexDir  string // absolute path to index/[version]
	Updated   bool   // true if an update was performed this run
}

// EnsureVersion fetches the latest release; if local version is missing or different,
// downloads strings.zip, extracts to data/[version], builds index to index/[version],
// and writes the version file. Returns the current version and paths.
func EnsureVersion(baseDir string) (*EnsureResult, error) {
	baseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve base dir: %w", err)
	}

	local, err := GetLocalVersion(baseDir)
	if err != nil {
		return nil, err
	}

	release, err := GetLatestRelease()
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}

	latest := VersionFromTag(release.TagName)
	zipURL := GetStringsZipURL(release)
	if zipURL == "" {
		return nil, fmt.Errorf("release %s has no asset %s", release.TagName, stringsZipName)
	}

	stringDir := filepath.Join(baseDir, "strings", latest)
	indexDir := filepath.Join(baseDir, "index", latest)

	if local != latest {
		if err := UpdateToVersion(baseDir, latest, zipURL); err != nil {
			return nil, err
		}
		return &EnsureResult{
			Version:   latest,
			StringDir: stringDir,
			IndexDir:  indexDir,
			Updated:   true,
		}, nil
	}

	// Local matches latest; ensure paths exist (re-download if missing)
	if _, err := os.Stat(stringDir); err != nil {
		if os.IsNotExist(err) {
			if err := UpdateToVersion(baseDir, latest, zipURL); err != nil {
				return nil, err
			}
			return &EnsureResult{
				Version:   latest,
				StringDir: stringDir,
				IndexDir:  indexDir,
				Updated:   true,
			}, nil
		}
		return nil, err
	}
	return &EnsureResult{
		Version:   local,
		StringDir: stringDir,
		IndexDir:  indexDir,
		Updated:   false,
	}, nil
}

func downloadFile(url string, dest string) error {
	log.Printf("Downloading %s ...", url)
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download status %d: %s", resp.StatusCode, string(body))
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func extractZip(zipPath string, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := filepath.Join(destDir, filepath.Clean(f.Name))
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			return err
		}
		out, err := os.Create(name)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
