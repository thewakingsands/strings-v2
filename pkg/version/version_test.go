package version

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

const testVersion = "publish-20260728-3b64471"

// baseDirLayout describes what to create under a fresh base dir.
type baseDirLayout struct {
	versionFile string // written to baseDir/version when not "-"
	stringDir   bool
	indexDir    bool
}

func writeLayout(t *testing.T, layout baseDirLayout) string {
	t.Helper()

	baseDir := t.TempDir()
	if layout.versionFile != "-" {
		path := filepath.Join(baseDir, "version")
		if err := os.WriteFile(path, []byte(layout.versionFile), 0o644); err != nil {
			t.Fatalf("write version file: %v", err)
		}
	}
	if layout.stringDir {
		if err := os.MkdirAll(filepath.Join(baseDir, "strings", testVersion), 0o755); err != nil {
			t.Fatalf("mkdir strings dir: %v", err)
		}
	}
	if layout.indexDir {
		if err := os.MkdirAll(filepath.Join(baseDir, "index", testVersion), 0o755); err != nil {
			t.Fatalf("mkdir index dir: %v", err)
		}
	}
	return baseDir
}

func TestResolveLocalVersion(t *testing.T) {
	cases := []struct {
		name    string
		layout  baseDirLayout
		wantErr bool
	}{
		{
			name:   "version file with both directories",
			layout: baseDirLayout{versionFile: testVersion + "\n", stringDir: true, indexDir: true},
		},
		{
			name:   "index only is enough",
			layout: baseDirLayout{versionFile: testVersion + "\n", indexDir: true},
		},
		{
			// LoadStore rebuilds the index from the strings, which needs no network.
			name:   "strings only is enough",
			layout: baseDirLayout{versionFile: testVersion + "\n", stringDir: true},
		},
		{
			name:   "version file without surrounding whitespace",
			layout: baseDirLayout{versionFile: testVersion, indexDir: true},
		},
		{
			name:    "neither directory exists",
			layout:  baseDirLayout{versionFile: testVersion + "\n"},
			wantErr: true,
		},
		{
			name:    "no version file",
			layout:  baseDirLayout{versionFile: "-", indexDir: true},
			wantErr: true,
		},
		{
			name:    "empty version file",
			layout:  baseDirLayout{versionFile: "", indexDir: true},
			wantErr: true,
		},
		{
			name:    "whitespace only version file",
			layout:  baseDirLayout{versionFile: "  \n\t", indexDir: true},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			baseDir := writeLayout(t, tc.layout)

			result, err := ResolveLocalVersion(baseDir)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("ResolveLocalVersion = %+v, want an error", result)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveLocalVersion: %v", err)
			}

			if result.Version != testVersion {
				t.Errorf("version = %q, want %q", result.Version, testVersion)
			}
			if want := filepath.Join(baseDir, "strings", testVersion); result.StringDir != want {
				t.Errorf("stringDir = %q, want %q", result.StringDir, want)
			}
			if want := filepath.Join(baseDir, "index", testVersion); result.IndexDir != want {
				t.Errorf("indexDir = %q, want %q", result.IndexDir, want)
			}
			if result.Updated {
				t.Error("updated = true, want false because nothing was downloaded")
			}
		})
	}
}

// ResolveLocalVersion must never report a version the file does not name, even when
// a newer directory is lying around from an update that died while indexing.
func TestResolveLocalVersionIgnoresNewerDirectories(t *testing.T) {
	baseDir := writeLayout(t, baseDirLayout{versionFile: testVersion + "\n", indexDir: true})
	partial := "publish-20260901-cafe123"
	if err := os.MkdirAll(filepath.Join(baseDir, "index", partial), 0o755); err != nil {
		t.Fatalf("mkdir partial index dir: %v", err)
	}

	result, err := ResolveLocalVersion(baseDir)
	if err != nil {
		t.Fatalf("ResolveLocalVersion: %v", err)
	}

	if result.Version != testVersion {
		t.Errorf("version = %q, want the recorded %q rather than the newer directory",
			result.Version, testVersion)
	}
}

func TestResolveLocalVersionReturnsAbsolutePaths(t *testing.T) {
	baseDir := writeLayout(t, baseDirLayout{versionFile: testVersion + "\n", indexDir: true})

	// A relative base dir is what main.go passes by default (-data defaults to "data").
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(wd, baseDir)
	if err != nil {
		t.Skipf("cannot express %s relative to %s", baseDir, wd)
	}

	result, err := ResolveLocalVersion(relative)
	if err != nil {
		t.Fatalf("ResolveLocalVersion(%q): %v", relative, err)
	}

	if !filepath.IsAbs(result.IndexDir) {
		t.Errorf("indexDir = %q, want an absolute path", result.IndexDir)
	}
	if want := filepath.Join(baseDir, "index", testVersion); result.IndexDir != want {
		t.Errorf("indexDir = %q, want %q", result.IndexDir, want)
	}
}

// A file where a directory belongs must not pass as usable data.
func TestResolveLocalVersionRejectsNonDirectories(t *testing.T) {
	baseDir := writeLayout(t, baseDirLayout{versionFile: testVersion + "\n"})
	for _, kind := range []string{"index", "strings"} {
		if err := os.MkdirAll(filepath.Join(baseDir, kind), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", kind, err)
		}
		path := filepath.Join(baseDir, kind, testVersion)
		if err := os.WriteFile(path, []byte("not a directory"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	if result, err := ResolveLocalVersion(baseDir); err == nil {
		t.Errorf("ResolveLocalVersion = %+v, want an error", result)
	}
}
