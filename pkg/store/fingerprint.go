package store

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	index "github.com/blevesearch/bleve_index_api"
)

// MappingFingerprint identifies the mapping this binary builds its index with.
//
// bleve keeps the mapping an index was built with inside the index, and every query
// against it is analyzed with that stored copy, never with buildItemIndexMapping. So a
// changed analyzer pointed at an existing index does nothing at all, and says nothing
// either. LoadStore compares this with the index's own fingerprint and refuses to serve
// when the two differ.
func MappingFingerprint() string {
	return fingerprintOf(mappingFacts(buildItemIndexMapping().(*mapping.IndexMappingImpl)))
}

// MappingMismatchError is an index built with a mapping other than this binary's.
// OnlyInIndex and OnlyInCode are the mapping facts that differ, so the message can say
// which field changed and not only that something did.
type MappingMismatchError struct {
	IndexDir    string
	Index       string
	Code        string
	OnlyInIndex []string
	OnlyInCode  []string
}

func (e *MappingMismatchError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "index %s was built with mapping %s, but this binary's is %s; "+
		"serving it would analyze every query the index's way and ignore this binary's. "+
		"Rebuild the index with this binary, or serve it with the one that built it.",
		e.IndexDir, e.Index, e.Code)
	for _, fact := range e.OnlyInIndex {
		fmt.Fprintf(&b, "\n  index:  %s", fact)
	}
	for _, fact := range e.OnlyInCode {
		fmt.Fprintf(&b, "\n  binary: %s", fact)
	}
	return b.String()
}

// verifyMapping refuses idx unless it was built with this binary's mapping.
func verifyMapping(idx bleve.Index, indexDir string) error {
	stored, ok := idx.Mapping().(*mapping.IndexMappingImpl)
	if !ok {
		return fmt.Errorf("index %s has a mapping of type %T, which cannot be fingerprinted", indexDir, idx.Mapping())
	}

	have := mappingFacts(stored)
	want := mappingFacts(buildItemIndexMapping().(*mapping.IndexMappingImpl))
	if slices.Equal(have, want) {
		return nil
	}
	return &MappingMismatchError{
		IndexDir:    indexDir,
		Index:       fingerprintOf(have),
		Code:        fingerprintOf(want),
		OnlyInIndex: factsMissingFrom(have, want),
		OnlyInCode:  factsMissingFrom(want, have),
	}
}

func fingerprintOf(facts []string) string {
	sum := sha256.Sum256([]byte(strings.Join(facts, "\n")))
	return hex.EncodeToString(sum[:8])
}

func factsMissingFrom(facts, other []string) []string {
	var missing []string
	for _, fact := range facts {
		if !slices.Contains(other, fact) {
			missing = append(missing, fact)
		}
	}
	return missing
}

// mappingFacts lists what m decides about indexing and querying, one fact per line,
// sorted.
//
// The facts are what bleve derives from the mapping rather than the mapping as written:
// the analyzer a field ends up with, not the name it happens to spell out. Hashing the
// serialized mapping would be simpler and would refuse every existing index the first
// time a bleve upgrade added a key to that JSON or wrote out a default differently, with
// nothing about the index having changed.
//
// What no mapping can show is the inside of a built-in analyzer. A bleve release that
// changes what "en" does keeps the name, and with it the fingerprint.
func mappingFacts(m *mapping.IndexMappingImpl) []string {
	facts := []string{fmt.Sprintf(
		"index type_field=%s default_type=%s default_field=%s index_dynamic=%t store_dynamic=%t docvalues_dynamic=%t scoring_model=%s",
		m.TypeField, m.DefaultType, m.DefaultField, m.IndexDynamic, m.StoreDynamic, m.DocValuesDynamic,
		cmp.Or(m.ScoringModel, index.DefaultScoringModel))}

	facts = appendDocumentFacts(facts, "_default", m.DefaultMapping, nil, m.DefaultAnalyzer)
	for name, dm := range m.TypeMapping {
		facts = appendDocumentFacts(facts, name, dm, nil, m.DefaultAnalyzer)
	}

	if ca := m.CustomAnalysis; ca != nil {
		kinds := map[string]map[string]map[string]interface{}{
			"char_filter":      ca.CharFilters,
			"tokenizer":        ca.Tokenizers,
			"token_map":        ca.TokenMaps,
			"token_filter":     ca.TokenFilters,
			"analyzer":         ca.Analyzers,
			"date_time_parser": ca.DateTimeParsers,
			"synonym_source":   ca.SynonymSources,
		}
		for kind, defs := range kinds {
			for name, config := range defs {
				// A custom component's definition is the code's own config, not something
				// bleve shapes, so its JSON is safe to compare, and encoding/json sorts the
				// keys. bleve stored this same config as JSON, so the error branch is only
				// reachable from a code mapping that could not have built an index either.
				encoded, err := json.Marshal(config)
				if err != nil {
					encoded = []byte(err.Error())
				}
				facts = append(facts, fmt.Sprintf("custom %s %s %s", kind, name, encoded))
			}
		}
	}

	slices.Sort(facts)
	return facts
}

// appendDocumentFacts adds what dm decides, found at path inside the mapping for type
// typeName: whether it is indexed, what happens to properties it does not map, and each
// field it does map.
//
// inherited is the analyzer a text field gets when it names none. bleve resolves it by
// walking from the type's root down to the field and keeping the last default analyzer
// it meets, and falls back to the index's when there is none.
func appendDocumentFacts(facts []string, typeName string, dm *mapping.DocumentMapping, path []string, inherited string) []string {
	if dm == nil {
		return facts
	}
	inherited = cmp.Or(dm.DefaultAnalyzer, inherited)
	at := strings.Join(append([]string{typeName}, path...), ".")

	facts = append(facts, fmt.Sprintf("document %s enabled=%t dynamic=%t analyzer=%s",
		at, dm.Enabled, dm.Dynamic, inherited))

	for _, fm := range dm.Fields {
		analyzer := "-"
		if fm.Type == "text" {
			analyzer = cmp.Or(fm.Analyzer, inherited)
		}
		facts = append(facts, fmt.Sprintf(
			"field %s as %s type=%s analyzer=%s index=%t store=%t docvalues=%t include_in_all=%t term_vectors=%t skip_freq_norm=%t",
			at, indexedFieldName(path, fm), fm.Type, analyzer,
			fm.Index, fm.Store, fm.DocValues, fm.IncludeInAll, fm.IncludeTermVectors, fm.SkipFreqNorm))
	}

	for name, child := range dm.Properties {
		facts = appendDocumentFacts(facts, typeName, child, append(slices.Clone(path), name), inherited)
	}
	return facts
}

// indexedFieldName is the name a field mapping indexes under, which a query has to use.
// A Name replaces only the last element of the path; this mirrors bleve's getFieldName.
func indexedFieldName(path []string, fm *mapping.FieldMapping) string {
	if fm.Name == "" {
		return strings.Join(path, ".")
	}
	if len(path) <= 1 {
		return fm.Name
	}
	return strings.Join(path[:len(path)-1], ".") + "." + fm.Name
}
