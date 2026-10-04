package store

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
)

// parseAdvancedQuery reads q as bleve's query string syntax: +must, -must_not,
// "phrases", field:value, *wildcards*, /regexps/, fuzzy~1, boost^2 and numeric ranges.
//
// A clause that names a field searches only that field. One that does not is spread
// over every language in langs, each copy weighted the way parseSearchQuery weights
// the plain search, and the copies are combined by dis_max. That is Elasticsearch's
// query_string with several default fields: +a +b needs a and b each to match in some
// language, not both in the same one, and -a drops a row that has a in any of them.
//
// Bleve itself would send such a clause to the _all field, which mixes the bigrams of
// the CJK columns with the single characters of the others, ignores the language order
// and has no idea of the sheet and id fields being keywords.
func parseAdvancedQuery(q string, langs []string, sheet string) (query.Query, error) {
	parsed, err := query.NewQueryStringQuery(q).Parse()
	if err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	nonLatin := isNonLatinQuery(strings.Join(unfieldedTexts(parsed, nil), " "))
	textQuery := spreadOverLanguages(parsed, langs, nonLatin)

	if sheet == "" {
		return textQuery, nil
	}

	sheetQuery := bleve.NewTermQuery(sheet)
	sheetQuery.SetField("sheet")

	return bleve.NewConjunctionQuery(textQuery, sheetQuery), nil
}

// spreadOverLanguages replaces every clause of q that names no field with a dis_max of
// one copy per language, and returns q with those replacements made.
func spreadOverLanguages(q query.Query, langs []string, nonLatin bool) query.Query {
	switch q := q.(type) {
	case *query.BooleanQuery:
		q.Must = spreadOverLanguages(q.Must, langs, nonLatin)
		q.Should = spreadOverLanguages(q.Should, langs, nonLatin)
		q.MustNot = spreadOverLanguages(q.MustNot, langs, nonLatin)
		q.Filter = spreadOverLanguages(q.Filter, langs, nonLatin)
		return q
	case *query.ConjunctionQuery:
		for i, conjunct := range q.Conjuncts {
			q.Conjuncts[i] = spreadOverLanguages(conjunct, langs, nonLatin)
		}
		return q
	case *query.DisjunctionQuery:
		for i, disjunct := range q.Disjuncts {
			q.Disjuncts[i] = spreadOverLanguages(disjunct, langs, nonLatin)
		}
		return q
	case query.FieldableQuery:
		if q.Field() != "" {
			return q
		}
		disjuncts := make([]query.Query, 0, len(langs))
		for i, lang := range langs {
			disjuncts = append(disjuncts, copyForLanguage(q, lang, languageBoost(i, lang, nonLatin)))
		}
		if len(disjuncts) == 1 {
			return disjuncts[0]
		}
		return newDisMaxQuery(disjuncts)
	default:
		// nil for the parts of a boolean query left empty, and anything else carries no
		// field to fill in.
		return q
	}
}

// copyForLanguage returns a copy of q searching lang, with q's own boost (from ^2 in
// the query) multiplied by boost.
//
// The copy is shallow, which is enough: SetField and SetBoost replace a field and a
// pointer rather than writing through them, so nothing the copies share is modified.
func copyForLanguage(q query.FieldableQuery, lang string, boost float64) query.Query {
	original := reflect.ValueOf(q)
	clone := reflect.New(original.Elem().Type())
	clone.Elem().Set(original.Elem())

	rv := clone.Interface().(query.FieldableQuery)
	rv.SetField(lang)
	if boostable, ok := rv.(query.BoostableQuery); ok && boost != 1 {
		boostable.SetBoost(boostable.Boost() * boost)
	}
	return rv
}

// unfieldedTexts collects the search text of every clause in q that names no field,
// the part of the query that ends up matched against the languages.
func unfieldedTexts(q query.Query, texts []string) []string {
	switch q := q.(type) {
	case *query.BooleanQuery:
		for _, child := range []query.Query{q.Must, q.Should, q.MustNot, q.Filter} {
			texts = unfieldedTexts(child, texts)
		}
	case *query.ConjunctionQuery:
		for _, conjunct := range q.Conjuncts {
			texts = unfieldedTexts(conjunct, texts)
		}
	case *query.DisjunctionQuery:
		for _, disjunct := range q.Disjuncts {
			texts = unfieldedTexts(disjunct, texts)
		}
	case *query.MatchQuery:
		if q.FieldVal == "" {
			texts = append(texts, q.Match)
		}
	case *query.MatchPhraseQuery:
		if q.FieldVal == "" {
			texts = append(texts, q.MatchPhrase)
		}
	case *query.WildcardQuery:
		if q.FieldVal == "" {
			texts = append(texts, q.Wildcard)
		}
	case *query.RegexpQuery:
		if q.FieldVal == "" {
			texts = append(texts, q.Regexp)
		}
	}
	return texts
}
