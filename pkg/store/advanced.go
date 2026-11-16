package store

import (
	"fmt"
	"reflect"
	"strings"

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
func parseAdvancedQuery(q string, langs []string, sheet SheetFilter) (query.Query, error) {
	parsed, err := query.NewQueryStringQuery(q).Parse()
	if err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	script := scriptOf(strings.Join(unfieldedTexts(parsed, nil), " "))
	textQuery := foldKanaClauses(spreadOverLanguages(parsed, langs, script))

	return sheet.apply(textQuery), nil
}

// foldKanaClauses lets every plain match clause on a kana folded field match either
// script, covering the copies spreadOverLanguages made as well as clauses that named the
// field themselves.
//
// Phrases stay exact, since quoting is how a user asks for the text as typed. Fuzzy,
// wildcard and regexp clauses are not analyzed into terms the way a match clause is, so
// they are left alone too. The query string parser only ever sets fuzziness through
// SetFuzziness, which is why a nonzero Fuzziness identifies every fuzzy clause.
func foldKanaClauses(q query.Query) query.Query {
	switch q := q.(type) {
	case *query.BooleanQuery:
		q.Must = foldKanaClauses(q.Must)
		q.Should = foldKanaClauses(q.Should)
		q.MustNot = foldKanaClauses(q.MustNot)
		q.Filter = foldKanaClauses(q.Filter)
		return q
	case *query.ConjunctionQuery:
		for i, conjunct := range q.Conjuncts {
			q.Conjuncts[i] = foldKanaClauses(conjunct)
		}
		return q
	case *query.DisjunctionQuery:
		for i, disjunct := range q.Disjuncts {
			q.Disjuncts[i] = foldKanaClauses(disjunct)
		}
		return q
	case *disMaxQuery:
		for i, disjunct := range q.disjuncts {
			q.disjuncts[i] = foldKanaClauses(disjunct)
		}
		return q
	case *query.MatchQuery:
		if foldsKana(q.FieldVal) && q.Fuzziness == 0 {
			return newKanaMatchQuery(q)
		}
		return q
	default:
		return q
	}
}

// spreadOverLanguages replaces every clause of q that names no field with a dis_max of
// one copy per language, and returns q with those replacements made. script is that of
// all such clauses together, as scriptOf reads it.
func spreadOverLanguages(q query.Query, langs []string, script queryScript) query.Query {
	switch q := q.(type) {
	case *query.BooleanQuery:
		q.Must = spreadOverLanguages(q.Must, langs, script)
		q.Should = spreadOverLanguages(q.Should, langs, script)
		q.MustNot = spreadOverLanguages(q.MustNot, langs, script)
		q.Filter = spreadOverLanguages(q.Filter, langs, script)
		return q
	case *query.ConjunctionQuery:
		for i, conjunct := range q.Conjuncts {
			q.Conjuncts[i] = spreadOverLanguages(conjunct, langs, script)
		}
		return q
	case *query.DisjunctionQuery:
		for i, disjunct := range q.Disjuncts {
			q.Disjuncts[i] = spreadOverLanguages(disjunct, langs, script)
		}
		return q
	case query.FieldableQuery:
		if q.Field() != "" {
			return q
		}
		disjuncts := make([]query.Query, 0, len(langs))
		for i, lang := range langs {
			copied := copyForLanguage(q, lang, languageBoost(i, lang, script))
			disjuncts = append(disjuncts, withQueryAnalyzer(copied, queryAnalyzer(lang, script)))
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

// withQueryAnalyzer has q analyze its text with analyzer, when q is a clause that analyzes
// its text at all and analyzer is not "". Wildcards, regular expressions and terms are
// matched as typed, so they are left alone.
func withQueryAnalyzer(q query.Query, analyzer string) query.Query {
	if analyzer == "" {
		return q
	}
	switch q := q.(type) {
	case *query.MatchQuery:
		q.Analyzer = analyzer
	case *query.MatchPhraseQuery:
		q.Analyzer = analyzer
	}
	return q
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
