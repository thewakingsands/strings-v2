package store

import (
	"context"
	"fmt"
	"slices"

	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	index "github.com/blevesearch/bleve_index_api"
)

// kanaFoldedLanguages name the fields where hiragana and katakana match each other.
// Only Japanese writes one word in either script; chs, tc and ko carry kana only as
// untranslated Japanese, and folding there would merely widen the noise.
var kanaFoldedLanguages = []string{"ja"}

func foldsKana(lang string) bool {
	return slices.Contains(kanaFoldedLanguages, lang)
}

// kanaVariantBoost weights a query term matched only in the script the user did not
// type. It has to be low enough that the typed script still wins and high enough that a
// name typed in the wrong script still beats rows matching a bigram or two as typed.
//
// Measured on publish-20260728-3b64471 with the ja field alone: at 1.0 a hiragana
// greeting got its katakana spelling mixed into the top ranks, and at 0.3 a long
// katakana name typed in hiragana lost the first page to rows sharing part of it. 0.5
// was the one value of the three that did both.
const kanaVariantBoost = 0.5

// The hiragana block U+3041–U+3096 and the katakana block U+30A1–U+30F6 line up one to
// one, as do the iteration marks ゝゞ and ヽヾ.
const (
	hiraganaFirst  = 0x3041
	hiraganaLast   = 0x3096
	hiraIterFirst  = 0x309D
	hiraIterLast   = 0x309E
	katakanaOffset = 0x60
)

// kanaCounterpart returns the same kana in the other script. ー and ・ are shared by
// both scripts, and ヷヸヹヺ, ヿ, ゟ and the small katakana extensions have no single
// code point counterpart, so those report false like every non-kana rune.
func kanaCounterpart(r rune) (rune, bool) {
	switch {
	case r >= hiraganaFirst && r <= hiraganaLast,
		r >= hiraIterFirst && r <= hiraIterLast:
		return r + katakanaOffset, true
	case r >= hiraganaFirst+katakanaOffset && r <= hiraganaLast+katakanaOffset,
		r >= hiraIterFirst+katakanaOffset && r <= hiraIterLast+katakanaOffset:
		return r - katakanaOffset, true
	}
	return 0, false
}

// kanaVariants returns term followed by every other spelling reached by switching any
// of its kana to the other script. The cjk analyzer emits bigrams, so a term carries at
// most two kana and at most four spellings.
func kanaVariants(term string) []string {
	variants := []string{""}
	for _, r := range term {
		other, ok := kanaCounterpart(r)
		next := make([]string, 0, 2*len(variants))
		for _, prefix := range variants {
			next = append(next, prefix+string(r))
			if ok {
				next = append(next, prefix+string(other))
			}
		}
		variants = next
	}
	return variants
}

// foldKana turns every katakana with a counterpart into hiragana, so two spellings of
// one word compare equal.
func foldKana(s string) string {
	runes := []rune(s)
	for i, r := range runes {
		if r >= hiraganaFirst+katakanaOffset {
			if other, ok := kanaCounterpart(r); ok {
				runes[i] = other
			}
		}
	}
	return string(runes)
}

// kanaMatchQuery is a match query that also accepts each term in the other kana script,
// scoring that spelling at kanaVariantBoost.
//
// It folds per term rather than per document or per query. Falling back only when the
// typed spelling matches nothing never fires, since a hiragana query nearly always
// shares some bigram with ordinary text; ranking every typed-script match first puts
// rows sharing one bigram as typed above rows matching all of them in the other script.
//
// The spellings of one term are combined by dis_max, not summed: a row holding a word
// in both scripts, as a katakana name followed by its hiragana reading does, would
// otherwise count it twice and outrank rows holding the name alone.
type kanaMatchQuery struct {
	match *query.MatchQuery
}

func newKanaMatchQuery(match *query.MatchQuery) *kanaMatchQuery {
	return &kanaMatchQuery{match: match}
}

func (q *kanaMatchQuery) SetBoost(b float64) { q.match.SetBoost(b) }
func (q *kanaMatchQuery) Boost() float64     { return q.match.Boost() }

// Searcher analyzes the text the way query.MatchQuery.Searcher does, so the terms agree
// with the index, and hands a query without kana straight to it: those score exactly as
// they did before folding existed.
func (q *kanaMatchQuery) Searcher(ctx context.Context, i index.IndexReader,
	m mapping.IndexMapping, options search.SearcherOptions,
) (search.Searcher, error) {
	field := q.match.FieldVal
	if field == "" {
		field = m.DefaultSearchField()
	}
	analyzerName := q.match.Analyzer
	if analyzerName == "" {
		analyzerName = m.AnalyzerNameForPath(field)
	}
	analyzer := m.AnalyzerNamed(analyzerName)
	if analyzer == nil {
		return nil, fmt.Errorf("no analyzer named '%s' registered", analyzerName)
	}

	tokens := analyzer.Analyze([]byte(q.match.Match))
	spellings := make([][]string, len(tokens))
	folded := false
	for i, token := range tokens {
		spellings[i] = kanaVariants(string(token.Term))
		folded = folded || len(spellings[i]) > 1
	}
	if !folded {
		return q.match.Searcher(ctx, i, m, options)
	}

	boost := q.match.BoostVal.Value()
	termQuery := func(term string, boost float64) query.Query {
		tq := query.NewTermQuery(term)
		tq.SetField(field)
		tq.SetBoost(boost)
		return tq
	}

	termQueries := make([]query.Query, len(tokens))
	for i, variants := range spellings {
		if len(variants) == 1 {
			termQueries[i] = termQuery(variants[0], boost)
			continue
		}
		disjuncts := make([]query.Query, len(variants))
		disjuncts[0] = termQuery(variants[0], boost)
		for j, variant := range variants[1:] {
			disjuncts[j+1] = termQuery(variant, boost*kanaVariantBoost)
		}
		termQueries[i] = newDisMaxQuery(disjuncts)
	}

	switch q.match.Operator {
	case query.MatchQueryOperatorOr:
		should := query.NewDisjunctionQuery(termQueries)
		should.SetMin(1)
		should.SetBoost(boost)
		return should.Searcher(ctx, i, m, options)
	case query.MatchQueryOperatorAnd:
		must := query.NewConjunctionQuery(termQueries)
		must.SetBoost(boost)
		return must.Searcher(ctx, i, m, options)
	default:
		return nil, fmt.Errorf("unhandled operator %d", q.match.Operator)
	}
}
