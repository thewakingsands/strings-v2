package store

import (
	"os"
	"path/filepath"
	"testing"
)

const loadStoreFixture = `[
  {"sheet":"Item","rowId":"1","values":{"chs":"洋葱","tc":"洋蔥","en":"onion","ja":"オニオン"}},
  {"sheet":"Item","rowId":"2","values":{"chs":"洋葱汤","tc":"洋蔥湯","en":"onion soup","ja":"オニオンスープ"}},
  {"sheet":"Addon","rowId":"3","values":{"chs":"胜","tc":"勝","en":"Victory","ja":"勝利"}}
]`

// writeFixture lays out a data directory the way the extracted strings.zip looks.
func writeFixture(t *testing.T) (dataDir, indexDir string) {
	t.Helper()

	baseDir := t.TempDir()
	dataDir = filepath.Join(baseDir, "strings")
	indexDir = filepath.Join(baseDir, "index")

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "strings-0000.json"),
		[]byte(loadStoreFixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	return dataDir, indexDir
}

func TestLoadStoreBuildsTheIndexThenServesIt(t *testing.T) {
	dataDir, indexDir := writeFixture(t)

	// Nothing on disk yet, so this has to build the index and then reopen it read only.
	st, err := LoadStore(dataDir, indexDir)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}

	result, err := st.Search("onion", []string{"en"}, "", 0, 10, []string{"chs", "en"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("total = %d, want 2 (%v)", result.Total, rowIDsOf(result.Items))
	}

	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Second time around the index is there, so this takes the plain read only path.
	reopened, err := LoadStore(dataDir, indexDir)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	t.Cleanup(func() {
		if err := reopened.Close(); err != nil {
			t.Errorf("close reopened: %v", err)
		}
	})

	again, err := reopened.Search("onion", []string{"en"}, "", 0, 10, []string{"chs", "en"})
	if err != nil {
		t.Fatalf("search after reopen: %v", err)
	}
	if again.Total != result.Total {
		t.Errorf("total after reopen = %d, want %d", again.Total, result.Total)
	}
}

// Serving read only is what lets several instances share one index directory. Opening
// it writable would take an exclusive lock and the second reader here would time out,
// so this pins the behaviour rather than the implementation detail behind it.
func TestLoadStoreAllowsSeveralReadersAtOnce(t *testing.T) {
	dataDir, indexDir := writeFixture(t)

	const readers = 3
	stores := make([]*Store, 0, readers)
	for i := range readers {
		st, err := LoadStore(dataDir, indexDir)
		if err != nil {
			t.Fatalf("reader %d could not open the index while %d other(s) had it: %v", i+1, i, err)
		}
		t.Cleanup(func() {
			if err := st.Close(); err != nil {
				t.Errorf("close reader: %v", err)
			}
		})
		stores = append(stores, st)
	}

	for i, st := range stores {
		result, err := st.Search("onion", []string{"en"}, "", 0, 10, []string{"chs", "en"})
		if err != nil {
			t.Fatalf("reader %d search: %v", i+1, err)
		}
		if result.Total != 2 {
			t.Errorf("reader %d total = %d, want 2 (%v)", i+1, result.Total, rowIDsOf(result.Items))
		}
	}
}
