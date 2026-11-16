package store

import (
	"os"
	"path/filepath"
	"slices"
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

// absentWindow is how far down a wantAbsent row is looked for: the first page, which
// is the only part of a result list most people ever read.
const absentWindow = 10

// rankedRow bounds where a row may sit without saying anything about the rows around
// it. wantTop pins an exact order and breaks when two equally good results swap; this
// says only "no worse than here", which is usually the claim actually being made.
type rankedRow struct {
	rowID   string
	maxRank int
}

// relevanceCase pins one query's result list from three directions.
//
// wantTop is prefix matched, for the places where the order itself is the point.
// wantRanked bounds a row's position without fixing it. wantTotal pins the exact
// recall, which catches a query change that quietly widens or narrows what matches.
//
// wantAbsent is the other half of that, and the reason the first three are not enough:
// a change that widens recall shows up in wantTotal as a bigger number, and a bigger
// number is not obviously wrong. Naming the rows that must stay off the first page
// turns the same change into a failure that reads as one. The rows worth naming are
// the ones a previous fix pushed down, so if the fix is undone they come straight back
// to where they were.
type relevanceCase struct {
	name       string
	query      string
	langs      []string
	sheet      string
	wantTotal  uint64
	wantTop    []string
	wantRanked []rankedRow
	wantAbsent []string
}

// limit is how deep to search: far enough to judge every claim the case makes.
func (tc relevanceCase) limit() int {
	limit := len(tc.wantTop) + 5
	for _, ranked := range tc.wantRanked {
		if ranked.maxRank > limit {
			limit = ranked.maxRank
		}
	}
	if len(tc.wantAbsent) > 0 && limit < absentWindow {
		limit = absentWindow
	}
	return limit
}

var relevanceCases = []relevanceCase{
	{
		// Reported from production: this used to fall to 13th once more than one
		// language was searched, behind rows whose columns all read "R-WIN".
		//
		// Addon#5092 is one of those rows. It now sits around 18th, far enough down
		// that a return to the old behaviour trips both this and the wantTop above it.
		name:       "sentence outranks untranslated ui strings",
		query:      "I'll never win at rate",
		langs:      []string{"chs", "en", "ja"},
		wantTotal:  10340,
		wantTop:    []string{"DefaultTalk#597050"},
		wantAbsent: []string{"Addon#5092"},
	},
	{
		// The quest step comes first; the item and the later quest that mention the
		// same errand belong just behind it, but which of the two leads is not
		// something worth failing a build over.
		name:      "quest line and its item rank above the rest",
		query:     "The merchant is overjoyed to receive his stolen merchandise",
		langs:     []string{"chs", "en", "ja"},
		wantTotal: 6400,
		wantTop:   []string{"quest/002/ClsGla050_00256#8"},
		wantRanked: []rankedRow{
			{"EventItem#2001813", 5},
			{"quest/020/JobDrk450_02057#27", 5},
		},
	},
	{
		// Reported from production: the head used to be rows whose en column was never
		// translated and still holds CJK (街人A, 大爆発, 大放電), matching on a single
		// stray character while the Chinese column's 魔晶石师傅 was buried. Damping the
		// languages that can only match character by character pushed all three past
		// rank 1000, and naming them here is what makes undoing that a test failure
		// rather than a larger wantTotal.
		//
		// Nothing is pinned at rank 1. Addon#15177 holds it by coincidence: its ja
		// column reads 大人気, so the bigram 大人 matches. Suppressing that needs phrase
		// or AND semantics, a separate decision, and stating the claim as "the Chinese
		// matches are near the top" means this case does not have to be re-pinned on
		// the day it lands.
		name:      "chinese match beats untranslated english column",
		query:     "师傅大人",
		langs:     []string{"chs", "en", "ja"},
		wantTotal: 6407,
		wantRanked: []rankedRow{
			{"PlaceName#1296", 5},
			{"Completion#1231", 5},
		},
		wantAbsent: []string{
			"ENpcResident#1007198",
			"Action#23942",
			"Action#37152",
		},
	},
	{
		// The other half of the same fix: damping is a multiplier, so searching one
		// language on its own keeps its order, and untranslated text stays findable.
		// The whole 街人 family has to come back, not only the three that lead.
		name:      "untranslated text is still findable in the en column",
		query:     "街人",
		langs:     []string{"en"},
		wantTotal: 35,
		wantTop: []string{
			"ENpcResident#1007198",
			"ENpcResident#1007201",
			"ENpcResident#1007202",
		},
		wantRanked: []rankedRow{
			{"ENpcResident#1007199", 6},
			{"ENpcResident#1007200", 6},
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
	{
		// A Latin query with a language that is not the first one: confirms the accented
		// alphabets are not mistaken for a non-Latin script and damped.
		name:      "german stays undamped",
		query:     "Zwiebel",
		langs:     []string{"de"},
		sheet:     "Item",
		wantTotal: 20,
		wantTop:   []string{"Item#3513", "Item#3279", "Item#36097"},
	},
}

var relevanceFields = []string{"chs", "tc", "en", "ja"}

// pinnedDataDir finds the data tree to read, resolving it the same way the external
// relevance eval harness does, so that one EVAL_DATA configures both.
//
// The fallback matters more than it looks. data/ is gitignored and runs to gigabytes,
// so it is never copied into a worktree, and a worktree is exactly where ranking gets
// changed. Without this the case set below skips silently in the one place it was
// written to be read.
func pinnedDataDir(t *testing.T) string {
	t.Helper()

	if fromEnv := os.Getenv("EVAL_DATA"); fromEnv != "" {
		return fromEnv
	}

	here, err := filepath.Abs("../../data")
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(here); err == nil && info.IsDir() {
		return here
	}

	// A linked worktree's .git is a file reading "gitdir: <main>/.git/worktrees/<name>".
	marker, err := os.ReadFile(filepath.Join(filepath.Dir(here), ".git"))
	if err != nil {
		return here
	}
	text := strings.TrimSpace(string(marker))
	gitDir, ok := strings.CutPrefix(text, "gitdir:")
	if !ok {
		return here
	}
	worktrees := filepath.Dir(strings.TrimSpace(gitDir))
	if filepath.Base(worktrees) != "worktrees" {
		return here
	}
	main := filepath.Join(filepath.Dir(filepath.Dir(worktrees)), "data")
	if info, err := os.Stat(main); err == nil && info.IsDir() {
		return main
	}
	return here
}

// openPinnedStore opens the real index read only, or skips when it cannot.
func openPinnedStore(t *testing.T) *Store {
	t.Helper()

	baseDir := pinnedDataDir(t)

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

	// Fails rather than skips. The index answers with the mapping it was built with, so
	// after a mapping change in code these cases would go on passing against the old
	// behaviour while appearing to measure the new one.
	if err := verifyMapping(idx, indexDir); err != nil {
		t.Fatal(err)
	}

	return &Store{index: idx}
}

func TestRelevanceOnPinnedData(t *testing.T) {
	st := openPinnedStore(t)

	for _, tc := range relevanceCases {
		t.Run(tc.name, func(t *testing.T) {
			limit := tc.limit()
			var sheet SheetFilter
			if tc.sheet != "" {
				sheet.Include = []string{tc.sheet}
			}
			result, err := st.Search(t.Context(), tc.query, tc.langs, sheet, 0, limit, relevanceFields)
			if err != nil {
				t.Fatalf("search: %v", err)
			}

			got := rowIDsOf(result.Items)
			where := func() string {
				return "\n  q=" + tc.query + " lang=" + strings.Join(tc.langs, ",") +
					" sheet=" + tc.sheet + "\n  got: " + strings.Join(got, " ")
			}

			if result.Total != tc.wantTotal {
				t.Errorf("total = %d, want %d%s", result.Total, tc.wantTotal, where())
			}

			if len(got) < len(tc.wantTop) {
				t.Fatalf("got %d results, want at least %d\n  got: %v", len(got), len(tc.wantTop), got)
			}
			for i, want := range tc.wantTop {
				if got[i] != want {
					t.Errorf("rank %d = %s, want %s%s", i+1, got[i], want, where())
				}
			}

			for _, ranked := range tc.wantRanked {
				rank := slices.Index(got, ranked.rowID) + 1
				if rank == 0 {
					t.Errorf("%s is not in the first %d results, want it by rank %d%s",
						ranked.rowID, limit, ranked.maxRank, where())
					continue
				}
				if rank > ranked.maxRank {
					t.Errorf("%s is at rank %d, want it by rank %d%s",
						ranked.rowID, rank, ranked.maxRank, where())
				}
			}

			for _, ref := range tc.wantAbsent {
				if rank := slices.Index(got, ref) + 1; rank > 0 && rank <= absentWindow {
					t.Errorf("%s is at rank %d and should not be in the first %d: "+
						"recall widened somewhere and took precision with it%s",
						ref, rank, absentWindow, where())
				}
			}
		})
	}
}
