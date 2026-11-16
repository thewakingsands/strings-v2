package store

import (
	"slices"
	"testing"

	"github.com/blevesearch/bleve/v2"
)

// kanaTestItems spell invented words in one script, the other, or both.
func kanaTestItems() []*Item {
	return []*Item{
		{Sheet: "Kana", RowID: "1", Index: 0, Values: map[string]string{"ja": "カタカナ"}},
		{Sheet: "Kana", RowID: "2", Index: 1, Values: map[string]string{"ja": "かたかな"}},
		// A katakana word followed by its hiragana reading, the way item names are stored.
		{Sheet: "Kana", RowID: "3", Index: 2, Values: map[string]string{"ja": "カタカナ\nかたかな"}},
		{Sheet: "Kana", RowID: "4", Index: 3, Values: map[string]string{"chs": "てすと", "ja": "テスト"}},
		{Sheet: "Kana", RowID: "5", Index: 4, Values: map[string]string{"ja": "にほんご"}},
		{Sheet: "Kana", RowID: "6", Index: 5, Values: map[string]string{"ja": "漢字", "en": "kanji"}},
		{Sheet: "Kana", RowID: "7", Index: 6, Values: map[string]string{"ja": "漢字カタカナ", "en": "kanji katakana"}},
	}
}

var kanaTestFields = []string{"chs", "ja", "en"}

func newKanaTestStore(t *testing.T) *Store {
	t.Helper()

	idx, err := bleve.NewMemOnly(buildItemIndexMapping())
	if err != nil {
		t.Fatalf("create in-memory index: %v", err)
	}
	t.Cleanup(func() {
		if err := idx.Close(); err != nil {
			t.Errorf("close index: %v", err)
		}
	})
	if err := indexItems(idx, kanaTestItems()); err != nil {
		t.Fatalf("index items: %v", err)
	}
	return &Store{index: idx}
}

func TestKanaCounterpart(t *testing.T) {
	pairs := [][2]rune{{'ぁ', 'ァ'}, {'ゖ', 'ヶ'}, {'ゝ', 'ヽ'}, {'ゞ', 'ヾ'}, {'ゔ', 'ヴ'}, {'か', 'カ'}}
	for _, pair := range pairs {
		if got, ok := kanaCounterpart(pair[0]); !ok || got != pair[1] {
			t.Errorf("kanaCounterpart(%q) = %q, %v; want %q", pair[0], got, ok, pair[1])
		}
		if got, ok := kanaCounterpart(pair[1]); !ok || got != pair[0] {
			t.Errorf("kanaCounterpart(%q) = %q, %v; want %q", pair[1], got, ok, pair[0])
		}
	}

	// Shared by both scripts, or with no single code point in the other one.
	for _, r := range []rune{'ー', '・', 'ヷ', 'ヺ', 'ヿ', 'ゟ', 'ㇰ', '\u3097', '漢', 'a', '０'} {
		if got, ok := kanaCounterpart(r); ok {
			t.Errorf("kanaCounterpart(%q) = %q, want none", r, got)
		}
	}
}

func TestKanaVariants(t *testing.T) {
	cases := []struct {
		term string
		want []string
	}{
		{"かた", []string{"かた", "かタ", "カた", "カタ"}},
		{"カタ", []string{"カタ", "カた", "かタ", "かた"}},
		{"字カ", []string{"字カ", "字か"}},
		{"ーカ", []string{"ーカ", "ーか"}},
		{"漢字", []string{"漢字"}},
		{"kanji", []string{"kanji"}},
	}
	for _, tc := range cases {
		if got := kanaVariants(tc.term); !slices.Equal(got, tc.want) {
			t.Errorf("kanaVariants(%q) = %q, want %q", tc.term, got, tc.want)
		}
	}
}

func TestFoldKana(t *testing.T) {
	if got, want := foldKana("カタカナとかたかなの漢字ーヽヷ"), "かたかなとかたかなの漢字ーゝヷ"; got != want {
		t.Errorf("foldKana = %q, want %q", got, want)
	}
}
