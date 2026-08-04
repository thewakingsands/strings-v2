package store

import (
	"log"
	"slices"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/highlight"
)

// termLocations finds where q matches inside text, both analyzed the same way.
//
// Running the field's own analyzer over both sides is what makes this agree with the
// index: quotes and punctuation are separators, so a query typed with "Software" matches
// a value stored with “Software”; en text is stemmed, so hardly matches hardliness; and
// cjk text is cut into bigrams. Comparing characters, as a browser would have to, gets
// all three wrong.
//
// analysis.Token carries byte offsets, the same units search.Location wants, so the
// offsets move across untouched.
func termLocations(analyzer analysis.Analyzer, q, text string) search.TermLocationMap {
	if analyzer == nil || q == "" || text == "" {
		return nil
	}

	wanted := make(map[string]struct{})
	for _, token := range analyzer.Analyze([]byte(q)) {
		wanted[string(token.Term)] = struct{}{}
	}
	if len(wanted) == 0 {
		return nil
	}

	var locations search.TermLocationMap
	for _, token := range analyzer.Analyze([]byte(text)) {
		term := string(token.Term)
		if _, ok := wanted[term]; !ok {
			continue
		}
		if locations == nil {
			locations = make(search.TermLocationMap)
		}
		locations.AddLocation(term, &search.Location{
			Pos:   uint64(token.Position),
			Start: uint64(token.Start),
			End:   uint64(token.End),
		})
	}

	return locations
}

// analyzerFor returns the analyzer the index actually uses for a field.
//
// Taken from the index's own mapping rather than buildItemIndexMapping so that a query is
// always analyzed the way the data was, even if the code's mapping has moved on.
func (s *Store) analyzerFor(m *mapping.IndexMappingImpl, lang string) analysis.Analyzer {
	if m == nil {
		return nil
	}
	return m.AnalyzerNamed(m.AnalyzerNameForPath(lang))
}

// addDisplayFragments gives the requested languages that took no part in the search a
// fragment of their own, so every column in a row is trimmed to a comparable length
// instead of one showing a snippet and the rest showing thousands of characters.
//
// Languages that did take part are left alone: bleve already fragmented them, and if one
// of them has no match on this row then it genuinely has none.
//
// This runs after the search, over one page of hits, so its cost follows the page size
// and not how much the query matched. Adding these languages to the query instead would
// cost what their recall costs, which for an English query against an English column is
// tens of thousands of documents.
func (s *Store) addDisplayFragments(hits search.DocumentMatchCollection, q string, langs, fields []string) {
	displayOnly := make([]string, 0, len(fields))
	for _, lang := range fields {
		if !slices.Contains(langs, lang) {
			displayOnly = append(displayOnly, lang)
		}
	}
	if len(displayOnly) == 0 || len(hits) == 0 {
		return
	}

	// The same highlighter Search asks for, so fragment size, centring on the match, the
	// <mark> tags and the HTML escaping are identical to the languages bleve handled.
	highlighter, err := bleve.Config.Cache.HighlighterNamed("html")
	if err != nil {
		log.Printf("no html highlighter, skipping display fragments: %v", err)
		return
	}

	indexMapping, _ := s.index.Mapping().(*mapping.IndexMappingImpl)

	analyzers := make(map[string]analysis.Analyzer, len(displayOnly))
	for _, lang := range displayOnly {
		analyzers[lang] = s.analyzerFor(indexMapping, lang)
	}

	for _, hit := range hits {
		s.addHitFragments(hit, q, displayOnly, analyzers, highlighter)
	}
}

func (s *Store) addHitFragments(
	hit *search.DocumentMatch,
	q string,
	displayOnly []string,
	analyzers map[string]analysis.Analyzer,
	highlighter highlight.Highlighter,
) {
	pending := make([]string, 0, len(displayOnly))
	for _, lang := range displayOnly {
		if text, ok := hit.Fields[lang].(string); ok && text != "" {
			pending = append(pending, lang)
		}
	}
	if len(pending) == 0 {
		return
	}

	// The highlighter reads the field values off the document, so it has to be loaded even
	// though the values are already on the hit.
	doc, err := s.index.Document(hit.ID)
	if err != nil || doc == nil {
		// Losing a preview is not worth failing the search over.
		return
	}

	for _, lang := range pending {
		text, _ := hit.Fields[lang].(string)
		if locations := termLocations(analyzers[lang], q, text); len(locations) > 0 {
			if hit.Locations == nil {
				hit.Locations = make(search.FieldTermLocationMap)
			}
			hit.Locations[lang] = locations
		}
		// Called even with no locations: that yields the head of the value, which is what
		// a column with nothing to highlight should show.
		highlighter.BestFragmentsInField(hit, doc, lang, 1)
	}
}
