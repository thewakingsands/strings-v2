package store

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/blevesearch/bleve/v2/search"
)

const completeEnglishValue = "wizard eggplant\nwizard eggplants\nA firm purple vegetable.\nWizard Eggplant"

// newTestHit builds a hit for Item#4785 whose en column matched and whose chs column
// did not, mirroring what Bleve hands back for a single language search.
func newTestHit() *search.DocumentMatch {
	return &search.DocumentMatch{
		ID: "Item@4785",
		Fields: map[string]any{
			"sheet": "Item",
			"id":    "4785",
			"index": float64(12),
			"en":    completeEnglishValue,
			"chs":   "巫师茄子",
		},
		Locations: search.FieldTermLocationMap{
			"en": search.TermLocationMap{
				"eggplant": search.Locations{{Pos: 2, Start: 7, End: 15}},
			},
		},
		Fragments: search.FieldFragmentMap{
			"en": []string{"…wizard <mark>eggplant</mark>\nwizard eggplants…"},
		},
	}
}

func TestFormatItemFromHitKeepsCompleteValues(t *testing.T) {
	item := formatItemFromHit(newTestHit())

	if item.Sheet != "Item" {
		t.Errorf("sheet = %q, want %q", item.Sheet, "Item")
	}
	if item.RowID != "4785" {
		t.Errorf("rowId = %q, want %q", item.RowID, "4785")
	}
	if item.Index != 12 {
		t.Errorf("index = %d, want 12", item.Index)
	}

	// The whole point of the change: a fragment must never stand in for the value.
	if item.Values["en"] != completeEnglishValue {
		t.Errorf("values[en] = %q, want the complete value %q", item.Values["en"], completeEnglishValue)
	}
	if item.Values["chs"] != "巫师茄子" {
		t.Errorf("values[chs] = %q, want %q", item.Values["chs"], "巫师茄子")
	}
	if strings.Contains(item.Values["en"], "…") || strings.Contains(item.Values["en"], "<mark>") {
		t.Errorf("values[en] = %q, want no fragment markers", item.Values["en"])
	}
}

func TestFormatItemFromHitReportsHighlightsSeparately(t *testing.T) {
	item := formatItemFromHit(newTestHit())

	want := "…wizard <mark>eggplant</mark>\nwizard eggplants…"
	if item.Highlights["en"] != want {
		t.Errorf("highlights[en] = %q, want %q", item.Highlights["en"], want)
	}
	if len(item.Highlights) != 1 {
		t.Errorf("highlights = %v, want only the en entry", item.Highlights)
	}
	if _, ok := item.Highlights["chs"]; ok {
		t.Errorf("highlights[chs] = %q, want no entry because chs has no fragment", item.Highlights["chs"])
	}
}

func TestFormatItemFromHitSkipsMetaFields(t *testing.T) {
	item := formatItemFromHit(newTestHit())

	for _, key := range metaFields {
		if _, ok := item.Values[key]; ok {
			t.Errorf("values contains meta field %q", key)
		}
	}
	if len(item.Values) != 2 {
		t.Errorf("values = %v, want only the language columns", item.Values)
	}
}

func TestFormatItemFromHitIgnoresFragmentWithoutMatch(t *testing.T) {
	hit := newTestHit()
	// A highlighted field that the query never matched still gets a fragment from
	// Bleve, but it is only the head of the value and carries no <mark>.
	hit.Fragments["chs"] = []string{"巫师茄…"}

	item := formatItemFromHit(hit)

	if got, ok := item.Highlights["chs"]; ok {
		t.Errorf("highlights[chs] = %q, want no entry because chs has no term locations", got)
	}
	if item.Values["chs"] != "巫师茄子" {
		t.Errorf("values[chs] = %q, want the complete value %q", item.Values["chs"], "巫师茄子")
	}
}

func TestFormatItemFromHitOmitsEmptyHighlights(t *testing.T) {
	hit := newTestHit()
	hit.Locations = nil
	hit.Fragments = nil

	item := formatItemFromHit(hit)

	if item.Highlights != nil {
		t.Errorf("highlights = %v, want nil so the field is dropped from JSON", item.Highlights)
	}

	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal item: %v", err)
	}
	if strings.Contains(string(encoded), "highlights") {
		t.Errorf("encoded item = %s, want no highlights key", encoded)
	}
}

func TestFormatItemFromHitHandlesEmptyFragmentList(t *testing.T) {
	hit := newTestHit()
	hit.Fragments["en"] = nil

	item := formatItemFromHit(hit)

	if got, ok := item.Highlights["en"]; ok {
		t.Errorf("highlights[en] = %q, want no entry for an empty fragment list", got)
	}
}
