package store

import (
	"encoding/json"
	"errors"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
	index "github.com/blevesearch/bleve_index_api"
)

func codeMapping(t *testing.T) *mapping.IndexMappingImpl {
	t.Helper()

	m, ok := buildItemIndexMapping().(*mapping.IndexMappingImpl)
	if !ok {
		t.Fatalf("mapping type = %T, want *mapping.IndexMappingImpl", buildItemIndexMapping())
	}
	return m
}

// itemField returns the field mapping of lang in m for a test to change. The four CJK
// languages share one *FieldMapping in buildItemIndexMapping, so it is copied first and
// a change to one of them stays a change to that one.
func itemField(m *mapping.IndexMappingImpl, lang string) *mapping.FieldMapping {
	property := m.TypeMapping["item"].Properties[lang]
	field := *property.Fields[0]
	property.Fields[0] = &field
	return &field
}

// storedForm is m the way an opened index hands it back: through the JSON bleve stores
// it as. Every level also gains a key this bleve has never heard of, the way a newer
// release might write one.
func storedForm(t *testing.T, m *mapping.IndexMappingImpl) *mapping.IndexMappingImpl {
	t.Helper()

	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal mapping: %v", err)
	}
	var tree map[string]any
	if err := json.Unmarshal(raw, &tree); err != nil {
		t.Fatalf("unmarshal mapping as a tree: %v", err)
	}

	tree["a_future_option"] = true
	item := tree["types"].(map[string]any)["item"].(map[string]any)
	item["a_future_option"] = true
	for _, property := range item["properties"].(map[string]any) {
		property := property.(map[string]any)
		property["a_future_option"] = true
		for _, field := range property["fields"].([]any) {
			field.(map[string]any)["a_future_option"] = true
		}
	}

	if raw, err = json.Marshal(tree); err != nil {
		t.Fatalf("marshal mapping tree: %v", err)
	}
	var stored *mapping.IndexMappingImpl
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("unmarshal mapping: %v", err)
	}
	return stored
}

// The point of fingerprinting what bleve derives rather than the JSON: a mapping that
// means the same thing has to keep the same fingerprint, or the first bleve upgrade
// refuses every index there is.
func TestMappingFingerprintIgnoresWhatDoesNotChangeTheIndex(t *testing.T) {
	want := mappingFacts(codeMapping(t))

	cases := []struct {
		name   string
		mapped func(t *testing.T) *mapping.IndexMappingImpl
	}{
		{"read back from the index, with keys this bleve does not know", func(t *testing.T) *mapping.IndexMappingImpl {
			return storedForm(t, codeMapping(t))
		}},
		{"the default scoring model spelled out", func(t *testing.T) *mapping.IndexMappingImpl {
			m := codeMapping(t)
			m.ScoringModel = index.DefaultScoringModel
			return m
		}},
		{"a field named after its own path", func(t *testing.T) *mapping.IndexMappingImpl {
			m := codeMapping(t)
			itemField(m, "en").Name = "en"
			return m
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mappingFacts(tc.mapped(t))
			if !slices.Equal(got, want) {
				t.Errorf("facts moved:\n  only now:    %v\n  only before: %v",
					factsMissingFrom(got, want), factsMissingFrom(want, got))
			}
		})
	}
}

// Each knob the fingerprint is meant to cover, turned on its own. Every one has to move
// the fingerprint, each to a value of its own, and keep it through the index's JSON:
// LoadStore compares the stored form with the code's, so a change that the trip into the
// index lost would never be noticed.
func TestMappingFingerprintChangesWithEveryKnob(t *testing.T) {
	addCustomAnalyzer := func(t *testing.T, m *mapping.IndexMappingImpl, filters ...string) {
		t.Helper()
		err := m.AddCustomAnalyzer("en_exact", map[string]interface{}{
			"type":          custom.Name,
			"tokenizer":     unicode.Name,
			"token_filters": filters,
		})
		if err != nil {
			t.Fatalf("add custom analyzer: %v", err)
		}
	}

	cases := []struct {
		name   string
		change func(t *testing.T, m *mapping.IndexMappingImpl)
	}{
		{"en analyzer", func(t *testing.T, m *mapping.IndexMappingImpl) { itemField(m, "en").Analyzer = keyword.Name }},
		{"ja analyzer", func(t *testing.T, m *mapping.IndexMappingImpl) { itemField(m, "ja").Analyzer = en.AnalyzerName }},
		{"de not indexed", func(t *testing.T, m *mapping.IndexMappingImpl) { itemField(m, "de").Index = false }},
		{"fr not stored", func(t *testing.T, m *mapping.IndexMappingImpl) { itemField(m, "fr").Store = false }},
		{"chs without docvalues", func(t *testing.T, m *mapping.IndexMappingImpl) { itemField(m, "chs").DocValues = false }},
		{"tc left out of _all", func(t *testing.T, m *mapping.IndexMappingImpl) { itemField(m, "tc").IncludeInAll = false }},
		{"ko without term vectors", func(t *testing.T, m *mapping.IndexMappingImpl) { itemField(m, "ko").IncludeTermVectors = false }},
		{"en without norms", func(t *testing.T, m *mapping.IndexMappingImpl) { itemField(m, "en").SkipFreqNorm = true }},
		{"en indexed under another name", func(t *testing.T, m *mapping.IndexMappingImpl) { itemField(m, "en").Name = "en_exact" }},
		{"a second field on the en path", func(t *testing.T, m *mapping.IndexMappingImpl) {
			exact := bleve.NewTextFieldMapping()
			exact.Name = "en_exact"
			exact.Analyzer = keyword.Name
			m.TypeMapping["item"].AddFieldMappingsAt("en", exact)
		}},
		{"a language added", func(t *testing.T, m *mapping.IndexMappingImpl) {
			m.TypeMapping["item"].AddFieldMappingsAt("cn", itemField(m, "chs"))
		}},
		{"a language dropped", func(t *testing.T, m *mapping.IndexMappingImpl) { delete(m.TypeMapping["item"].Properties, "fr") }},
		{"item mapping not dynamic", func(t *testing.T, m *mapping.IndexMappingImpl) { m.TypeMapping["item"].Dynamic = false }},
		{"default analyzer", func(t *testing.T, m *mapping.IndexMappingImpl) { m.DefaultAnalyzer = keyword.Name }},
		{"default type", func(t *testing.T, m *mapping.IndexMappingImpl) { m.DefaultType = "row" }},
		{"dynamic fields not indexed", func(t *testing.T, m *mapping.IndexMappingImpl) { m.IndexDynamic = false }},
		{"dynamic fields not stored", func(t *testing.T, m *mapping.IndexMappingImpl) { m.StoreDynamic = false }},
		{"dynamic fields without docvalues", func(t *testing.T, m *mapping.IndexMappingImpl) { m.DocValuesDynamic = false }},
		{"scoring model", func(t *testing.T, m *mapping.IndexMappingImpl) { m.ScoringModel = index.BM25Scoring }},
		{"a custom analyzer", func(t *testing.T, m *mapping.IndexMappingImpl) {
			addCustomAnalyzer(t, m, lowercase.Name)
		}},
		{"the same custom analyzer with another filter", func(t *testing.T, m *mapping.IndexMappingImpl) {
			addCustomAnalyzer(t, m, en.PossessiveName, lowercase.Name)
		}},
	}

	code := MappingFingerprint()
	seen := map[string]string{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := codeMapping(t)
			tc.change(t, m)

			got := fingerprintOf(mappingFacts(m))
			if got == code {
				t.Fatalf("fingerprint unchanged at %s", got)
			}
			if other, ok := seen[got]; ok {
				t.Errorf("fingerprint %s is also what %q gives", got, other)
			}
			seen[got] = tc.name

			if stored := fingerprintOf(mappingFacts(storedForm(t, m))); stored != got {
				t.Errorf("fingerprint read back from the index = %s, want %s", stored, got)
			}
		})
	}
}

// The failure this card exists for: the en analyzer swapped in code, then pointed at an
// index built before the swap. Without the check this serves, and serves the old analyzer.
func TestLoadStoreRefusesAnIndexBuiltWithAnotherMapping(t *testing.T) {
	dataDir, indexDir := writeFixture(t)

	old := codeMapping(t)
	itemField(old, "en").Analyzer = keyword.Name
	idx, err := bleve.New(indexDir, old)
	if err != nil {
		t.Fatalf("build index with the old mapping: %v", err)
	}
	items, err := loadFile(filepath.Join(dataDir, "strings-0000.json"))
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	if err := indexItems(idx, items); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
	if err := idx.Close(); err != nil {
		t.Fatalf("close index: %v", err)
	}

	st, err := LoadStore(dataDir, indexDir)
	if err == nil {
		st.Close()
		t.Fatal("LoadStore served an index built with another mapping")
	}
	var mismatch *MappingMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("err = %v, want a *MappingMismatchError", err)
	}
	if mismatch.Code != MappingFingerprint() {
		t.Errorf("code fingerprint = %s, want %s", mismatch.Code, MappingFingerprint())
	}
	if len(mismatch.OnlyInIndex) != 1 || !strings.Contains(mismatch.OnlyInIndex[0], "as en type=text analyzer=keyword ") {
		t.Errorf("only in index = %q, want the en field with the keyword analyzer", mismatch.OnlyInIndex)
	}
	if len(mismatch.OnlyInCode) != 1 || !strings.Contains(mismatch.OnlyInCode[0], "as en type=text analyzer=en ") {
		t.Errorf("only in code = %q, want the en field with the en analyzer", mismatch.OnlyInCode)
	}

	// A refused index is one somebody is about to rebuild, which takes an exclusive lock.
	// If refusing leaked the shared one, that would wait out the timeout and fail.
	writable, err := bleve.OpenUsing(indexDir, map[string]interface{}{"bolt_timeout": "1s"})
	if err != nil {
		t.Fatalf("refused index is still locked: %v", err)
	}
	if err := writable.Close(); err != nil {
		t.Fatalf("close writable index: %v", err)
	}
}

// The mapping has index_dynamic on, so a field added to ItemDocument without a mapping
// of its own is no error anywhere: bleve indexes it with the standard analyzer, and the
// fingerprint stays put because the mapping did not change. A mapping for a field the
// document lacks fails as quietly the other way, matching nothing. This is the one place
// either shows up.
func TestEveryItemDocumentFieldHasItsOwnMapping(t *testing.T) {
	var documentFields []string
	docType := reflect.TypeFor[ItemDocument]()
	for i := range docType.NumField() {
		field := docType.Field(i)
		// bleve names a struct field by its json tag and pays no attention to omitempty.
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		switch name {
		case "-":
			continue
		case "":
			name = field.Name
		}
		documentFields = append(documentFields, name)
	}
	slices.Sort(documentFields)

	item := codeMapping(t).TypeMapping["item"]
	mapped := slices.Sorted(maps.Keys(item.Properties))
	if !slices.Equal(documentFields, mapped) {
		t.Errorf("ItemDocument fields %v, mapped fields %v", documentFields, mapped)
	}
	for name, property := range item.Properties {
		if len(property.Fields) == 0 {
			t.Errorf("%s has a property mapping but no field mapping, so it is indexed dynamically", name)
		}
	}
}
