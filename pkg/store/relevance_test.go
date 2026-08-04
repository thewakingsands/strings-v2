package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pinnedDataVersion is the data these expectations were recorded against. Ranking and
// totals may legitimately change when the game data does, so a different version skips
// rather than fails: rerun the queries, eyeball the results, then update both this
// constant and the expectations below together.
//
// Read straight from the version file instead of through pkg/version, which imports
// this package.
const pinnedDataVersion = "publish-20260728-3b64471"

// relevanceCase pins the head of a result list. wantTop is prefix matched, so results
// beyond it are free to move; wantTotal pins the exact recall, which catches a query
// change that quietly widens or narrows what matches.
type relevanceCase struct {
	name      string
	query     string
	langs     []string
	sheet     string
	wantTotal uint64
	wantTop   []string
}

var relevanceCases = []relevanceCase{
	{
		// Reported from production: this used to fall to 13th once more than one
		// language was searched, behind rows whose columns all read "R-WIN".
		name:      "sentence outranks untranslated ui strings",
		query:     "I'll never win at rate",
		langs:     []string{"chs", "en", "ja"},
		wantTotal: 10346,
		wantTop:   []string{"DefaultTalk#597050"},
	},
	{
		name:      "quest line and its item rank above the rest",
		query:     "The merchant is overjoyed to receive his stolen merchandise",
		langs:     []string{"chs", "en", "ja"},
		wantTotal: 6902,
		wantTop: []string{
			"quest/002/ClsGla050_00256#8",
			"EventItem#2001813",
			"quest/020/JobDrk450_02057#27",
		},
	},
	{
		name:      "cjk single language",
		query:     "圆葱",
		langs:     []string{"chs"},
		sheet:     "Item",
		wantTotal: 2,
		wantTop:   []string{"Item#8183", "Item#8166"},
	},
	{
		// Same result as the single language case above: adding languages must not
		// disturb a ranking that one language already gets right.
		name:      "cjk across languages",
		query:     "圆葱",
		langs:     []string{"chs", "en", "ja"},
		sheet:     "Item",
		wantTotal: 2,
		wantTop:   []string{"Item#8183", "Item#8166"},
	},
	{
		// The row whose en value is long enough to be truncated in a highlight.
		name:      "long value with an apostrophe",
		query:     "wives",
		langs:     []string{"en"},
		sheet:     "Item",
		wantTotal: 2,
		wantTop:   []string{"Item#4788", "Item#37640"},
	},
}

var relevanceFields = []string{"chs", "tc", "en", "ja"}

// openPinnedStore opens the real index read only, or skips when it cannot.
func openPinnedStore(t *testing.T) *Store {
	t.Helper()

	baseDir, err := filepath.Abs("../../data")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(baseDir, "version"))
	if err != nil {
		t.Skipf("no local data at %s: %v", baseDir, err)
	}
	if local := strings.TrimSpace(string(raw)); local != pinnedDataVersion {
		t.Skipf("local data is %s but these expectations were pinned against %s; "+
			"rerun the queries, review the results, then update pinnedDataVersion and the expectations",
			local, pinnedDataVersion)
	}

	indexDir := filepath.Join(baseDir, "index", pinnedDataVersion)
	if _, err := os.Stat(indexDir); err != nil {
		t.Skipf("no index at %s: %v", indexDir, err)
	}

	// Read only opens share the lock, so this coexists with a running server. It still
	// fails when something holds a write lock, such as an index build in progress, and
	// a short wait keeps that case from stalling the suite before it skips.
	idx, err := openIndexReadOnly(indexDir, "3s")
	if err != nil {
		t.Skipf("cannot open %s read only, something may hold a write lock on it: %v", indexDir, err)
	}
	t.Cleanup(func() {
		if err := idx.Close(); err != nil {
			t.Errorf("close index: %v", err)
		}
	})

	return &Store{index: idx}
}

func TestRelevanceOnPinnedData(t *testing.T) {
	st := openPinnedStore(t)

	for _, tc := range relevanceCases {
		t.Run(tc.name, func(t *testing.T) {
			limit := len(tc.wantTop) + 5
			result, err := st.Search(tc.query, tc.langs, tc.sheet, 0, limit, relevanceFields)
			if err != nil {
				t.Fatalf("search: %v", err)
			}

			got := rowIDsOf(result.Items)
			if result.Total != tc.wantTotal {
				t.Errorf("total = %d, want %d\n  q=%q lang=%s sheet=%q\n  got: %v",
					result.Total, tc.wantTotal, tc.query, strings.Join(tc.langs, ","), tc.sheet, got)
			}

			if len(got) < len(tc.wantTop) {
				t.Fatalf("got %d results, want at least %d\n  got: %v", len(got), len(tc.wantTop), got)
			}
			for i, want := range tc.wantTop {
				if got[i] != want {
					t.Errorf("rank %d = %s, want %s\n  q=%q lang=%s\n  got: %v\n  want prefix: %v",
						i+1, got[i], want, tc.query, strings.Join(tc.langs, ","), got, tc.wantTop)
				}
			}
		})
	}
}
