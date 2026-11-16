package store

import (
	"context"
	"fmt"

	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	index "github.com/blevesearch/bleve_index_api"
)

// termsMatchQuery is bleve's match query with two changes: a term the text repeats is
// searched once, and with foldKana each term also matches in the other kana script.
//
// A repeated term carries the weight of all its repeats, so it adds to a row's score what
// the repeats added together; only bleve's coord factor, matched terms over all terms,
// now counts each term once. Searching it once is what keeps a pasted paragraph inside the search
// deadline: the terms it repeats are its commonest, and each repeat walked the same long
// postings list again. A 1000 character Japanese line repeats about a third of its
// bigrams and took 8 s against the ja column, 2.9 s merged; 2000 characters went from
// past the deadline to 4.9 s. Measured with the relevance eval on
// publish-20260728-3b64471, merging moved 5 cases, 3 up and 2 down, and no metric the
// wrong way in any langmode.
//
// Kana folding works per term rather than per document or per query. Falling back only
// when the typed spelling matches nothing never fires, since a hiragana query nearly
// always shares some bigram with ordinary text; ranking every typed-script match first
// puts rows sharing one bigram as typed above rows matching all of them in the other
// script. The other spelling scores at kanaVariantBoost, and the spellings of one term
// are combined by dis_max, not summed: a row holding a word in both scripts, as a
// katakana name followed by its hiragana reading does, would otherwise count it twice and
// outrank rows holding the name alone.
type termsMatchQuery struct {
	match    *query.MatchQuery
	foldKana bool
}

func newTermsMatchQuery(match *query.MatchQuery, foldKana bool) *termsMatchQuery {
	return &termsMatchQuery{match: match, foldKana: foldKana}
}

func (q *termsMatchQuery) SetBoost(b float64) { q.match.SetBoost(b) }
func (q *termsMatchQuery) Boost() float64     { return q.match.Boost() }

// Searcher analyzes the text the way query.MatchQuery.Searcher does, so the terms agree
// with the index, and hands a text with no repeated term and nothing to fold straight to
// it: those score exactly as they would without this type.
func (q *termsMatchQuery) Searcher(ctx context.Context, i index.IndexReader,
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
	terms, repeats := countTerms(tokens)
	spellings := make([][]string, len(terms))
	folded := false
	for i, term := range terms {
		spellings[i] = []string{term}
		if q.foldKana {
			spellings[i] = kanaVariants(term)
		}
		folded = folded || len(spellings[i]) > 1
	}
	if !folded && len(terms) == len(tokens) {
		return q.match.Searcher(ctx, i, m, options)
	}

	boost := q.match.BoostVal.Value()
	termQuery := func(term string, boost float64) query.Query {
		tq := query.NewTermQuery(term)
		tq.SetField(field)
		tq.SetBoost(boost)
		return tq
	}

	termQueries := make([]query.Query, len(terms))
	for i, variants := range spellings {
		weight := boost * float64(repeats[i])
		if len(variants) == 1 {
			termQueries[i] = termQuery(variants[0], weight)
			continue
		}
		disjuncts := make([]query.Query, len(variants))
		disjuncts[0] = termQuery(variants[0], weight)
		for j, variant := range variants[1:] {
			disjuncts[j+1] = termQuery(variant, weight*kanaVariantBoost)
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

// countTerms returns the distinct terms of tokens in the order they first appear, and
// how many times each appears.
func countTerms(tokens analysis.TokenStream) (terms []string, repeats []int) {
	seen := make(map[string]int, len(tokens))
	for _, token := range tokens {
		term := string(token.Term)
		if at, ok := seen[term]; ok {
			repeats[at]++
			continue
		}
		seen[term] = len(terms)
		terms = append(terms, term)
		repeats = append(repeats, 1)
	}
	return terms, repeats
}
