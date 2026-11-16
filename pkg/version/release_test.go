package version

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// Synthetic, in the shape of the ixion export; never game text.
const releaseFixture = `[{"sheet":"Test","rowId":"1","values":{"en":"fixture row"}}]`

const newerVersion = "publish-20990101-0000000"

// releaseHits counts requests per endpoint of a fake GitHub.
type releaseHits struct {
	latest atomic.Int32
	zip    atomic.Int32
	other  atomic.Int32
}

func (h *releaseHits) total() int32 {
	return h.latest.Load() + h.zip.Load() + h.other.Load()
}

// useReleaseServer points latestReleaseURL at a test server for the duration of t.
func useReleaseServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(handler)
	saved := latestReleaseURL
	latestReleaseURL = srv.URL + "/releases/latest"
	t.Cleanup(func() {
		latestReleaseURL = saved
		srv.Close()
	})
	return srv
}

// fakeGitHub serves /releases/latest naming tag, with strings.zip pointing back at
// itself; the zip holds the given files. It swaps latestReleaseURL for the test.
func fakeGitHub(t *testing.T, tag string, files map[string]string) *releaseHits {
	t.Helper()

	zipBytes := buildZip(t, files)
	hits := &releaseHits{}
	var srv *httptest.Server
	srv = useReleaseServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			hits.latest.Add(1)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(GitHubRelease{
				TagName: tag,
				Assets: []ReleaseAsset{
					{Name: stringsZipName, BrowserDownloadURL: srv.URL + "/" + stringsZipName},
				},
			})
		case "/" + stringsZipName:
			hits.zip.Add(1)
			w.Write(zipBytes)
		default:
			hits.other.Add(1)
			http.NotFound(w, r)
		}
	}))
	return hits
}

// failingGitHub answers /releases/latest with a 500.
func failingGitHub(t *testing.T) *releaseHits {
	t.Helper()

	hits := &releaseHits{}
	useReleaseServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/releases/latest" {
			hits.latest.Add(1)
		} else {
			hits.other.Add(1)
		}
		http.Error(w, "unavailable", http.StatusInternalServerError)
	}))
	return hits
}

// forbiddenGitHub fails the test on any request at all.
func forbiddenGitHub(t *testing.T) *releaseHits {
	t.Helper()

	hits := &releaseHits{}
	useReleaseServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.other.Add(1)
		t.Errorf("unexpected request %s", r.URL)
		http.Error(w, "forbidden", http.StatusInternalServerError)
	}))
	return hits
}

func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func fullLayout(t *testing.T) string {
	t.Helper()
	return writeLayout(t, baseDirLayout{versionFile: testVersion + "\n", stringDir: true, indexDir: true})
}

func readVersionFile(t *testing.T, baseDir string) string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join(baseDir, "version"))
	if err != nil {
		t.Fatalf("read version file: %v", err)
	}
	return string(b)
}

func TestGetLatestReleaseReadsTheInjectedURL(t *testing.T) {
	fakeGitHub(t, "publish/20990101-0000000", nil)

	release, err := GetLatestRelease()
	if err != nil {
		t.Fatalf("GetLatestRelease: %v", err)
	}
	if release.TagName != "publish/20990101-0000000" {
		t.Errorf("tag = %q", release.TagName)
	}
	zipURL := GetStringsZipURL(release)
	if !strings.HasPrefix(zipURL, "http://127.0.0.1:") {
		t.Errorf("zip url = %q, want the test server", zipURL)
	}
}

func TestEnsureVersionSkipsDownloadWhenCurrent(t *testing.T) {
	baseDir := fullLayout(t)
	hits := fakeGitHub(t, testVersion, nil)

	result, err := EnsureVersion(baseDir)
	if err != nil {
		t.Fatalf("EnsureVersion: %v", err)
	}
	if result.Updated {
		t.Error("Updated = true, want false")
	}
	if result.Version != testVersion {
		t.Errorf("version = %q, want %q", result.Version, testVersion)
	}
	if n := hits.zip.Load(); n != 0 {
		t.Errorf("zip requested %d times, want 0", n)
	}
}

func TestEnsureVersionDownloadsANewerRelease(t *testing.T) {
	baseDir := fullLayout(t)
	hits := fakeGitHub(t, newerVersion, map[string]string{"strings-0000.json": releaseFixture})

	result, err := EnsureVersion(baseDir)
	if err != nil {
		t.Fatalf("EnsureVersion: %v", err)
	}
	if !result.Updated {
		t.Error("Updated = false, want true")
	}
	if result.Version != newerVersion {
		t.Errorf("version = %q, want %q", result.Version, newerVersion)
	}
	if got := readVersionFile(t, baseDir); got != newerVersion+"\n" {
		t.Errorf("version file = %q, want %q", got, newerVersion+"\n")
	}
	for _, dir := range []string{"strings", "index"} {
		if !dirExists(filepath.Join(baseDir, dir, newerVersion)) {
			t.Errorf("%s/%s missing", dir, newerVersion)
		}
	}
	if n := hits.zip.Load(); n != 1 {
		t.Errorf("zip requested %d times, want 1", n)
	}
}

func TestEnsureVersionReportsAFailedCheck(t *testing.T) {
	baseDir := fullLayout(t)
	failingGitHub(t)

	if _, err := EnsureVersion(baseDir); err == nil {
		t.Fatal("EnsureVersion succeeded, want an error")
	}
	if got := readVersionFile(t, baseDir); got != testVersion+"\n" {
		t.Errorf("version file = %q, want it untouched", got)
	}
}

func TestResolveStartupSkipsTheCheck(t *testing.T) {
	baseDir := fullLayout(t)
	hits := forbiddenGitHub(t)

	result, err := ResolveStartup(baseDir, true)
	if err != nil {
		t.Fatalf("ResolveStartup: %v", err)
	}
	if result.Version != testVersion || result.Updated {
		t.Errorf("result = %+v, want local %s, not updated", result, testVersion)
	}
	if n := hits.total(); n != 0 {
		t.Errorf("%d requests, want none", n)
	}
}

// Without local data the switch must refuse, not quietly fall back to downloading.
func TestResolveStartupSkipWithoutLocalData(t *testing.T) {
	baseDir := t.TempDir()
	hits := forbiddenGitHub(t)

	if _, err := ResolveStartup(baseDir, true); err == nil {
		t.Fatal("ResolveStartup succeeded on an empty base dir, want an error")
	}
	if n := hits.total(); n != 0 {
		t.Errorf("%d requests, want none", n)
	}
}

func TestResolveStartupFallsBackWhenTheCheckFails(t *testing.T) {
	baseDir := fullLayout(t)
	hits := failingGitHub(t)

	result, err := ResolveStartup(baseDir, false)
	if err != nil {
		t.Fatalf("ResolveStartup: %v", err)
	}
	if result.Version != testVersion || result.Updated {
		t.Errorf("result = %+v, want local %s, not updated", result, testVersion)
	}
	if n := hits.latest.Load(); n != 1 {
		t.Errorf("release checked %d times, want 1", n)
	}
}
