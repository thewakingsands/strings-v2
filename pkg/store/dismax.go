package store

import (
	"context"
	"fmt"
	"math"
	"reflect"

	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/blevesearch/bleve/v2/size"
	index "github.com/blevesearch/bleve_index_api"
)

// disMaxQuery scores a document by its best matching clause instead of by the sum of
// all of them, the way Elasticsearch's dis_max does. Bleve only offers a disjunction,
// which scores sum(clauses) * matchedClauses/totalClauses, and neither half of that
// suits searching the same text in several languages:
//
//   - Summing rewards a row for matching in many languages. A UI string nobody
//     translated, identical in every column, then collects a score per language and
//     buries a real sentence that only its own language can match.
//   - The coord factor punishes matching in one language, but that is the normal case:
//     someone typing English only ever matches the English column.
//
// Both belong at the term level, where matching more of the query words really does
// mean more relevant, and that is where the wrapped match queries still apply them.
type disMaxQuery struct {
	disjuncts []query.Query
}

func newDisMaxQuery(disjuncts []query.Query) *disMaxQuery {
	return &disMaxQuery{disjuncts: disjuncts}
}

func (q *disMaxQuery) Searcher(ctx context.Context, i index.IndexReader,
	m mapping.IndexMapping, options search.SearcherOptions,
) (search.Searcher, error) {
	searchers := make([]search.Searcher, 0, len(q.disjuncts))
	for _, disjunct := range q.disjuncts {
		sub, err := disjunct.Searcher(ctx, i, m, options)
		if err != nil {
			for _, opened := range searchers {
				_ = opened.Close()
			}
			return nil, err
		}
		if sub == nil {
			continue
		}
		searchers = append(searchers, sub)
	}

	return newDisMaxSearcher(searchers, options), nil
}

func (q *disMaxQuery) Validate() error {
	if len(q.disjuncts) == 0 {
		return fmt.Errorf("dis_max query has no clauses")
	}
	for _, disjunct := range q.disjuncts {
		if validatable, ok := disjunct.(query.ValidatableQuery); ok {
			if err := validatable.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

var reflectStaticSizeDisMaxSearcher int

func init() {
	var ds disMaxSearcher
	reflectStaticSizeDisMaxSearcher = int(reflect.TypeOf(ds).Size())
}

// disMaxSearcher walks its children the way bleve's DisjunctionSliceSearcher does,
// only the scoring differs. The traversal is deliberately the same: keeping one current
// match per child and repeatedly consuming whichever share the lowest document id is
// what makes a disjunction over sorted postings correct.
type disMaxSearcher struct {
	searchers []search.Searcher
	explain   bool

	currs        []*search.DocumentMatch
	matching     []*search.DocumentMatch
	matchingIdxs []int
	initialized  bool
}

func newDisMaxSearcher(searchers []search.Searcher,
	options search.SearcherOptions,
) *disMaxSearcher {
	s := &disMaxSearcher{
		searchers:    searchers,
		explain:      options.Explain,
		currs:        make([]*search.DocumentMatch, len(searchers)),
		matching:     make([]*search.DocumentMatch, len(searchers)),
		matchingIdxs: make([]int, len(searchers)),
	}
	s.computeQueryNorm()
	return s
}

func (s *disMaxSearcher) computeQueryNorm() {
	sumOfSquaredWeights := 0.0
	for _, searcher := range s.searchers {
		sumOfSquaredWeights += searcher.Weight()
	}
	if sumOfSquaredWeights == 0 {
		return
	}

	queryNorm := 1.0 / math.Sqrt(sumOfSquaredWeights)
	for _, searcher := range s.searchers {
		searcher.SetQueryNorm(queryNorm)
	}
}

func (s *disMaxSearcher) initSearchers(ctx *search.SearchContext) error {
	var err error
	for i, searcher := range s.searchers {
		if s.currs[i] != nil {
			ctx.DocumentMatchPool.Put(s.currs[i])
		}
		s.currs[i], err = searcher.Next(ctx)
		if err != nil {
			return err
		}
	}

	s.updateMatches()
	s.initialized = true
	return nil
}

// updateMatches collects every child sitting on the lowest document id.
func (s *disMaxSearcher) updateMatches() {
	matching := s.matching[:0]
	matchingIdxs := s.matchingIdxs[:0]

	for i := 0; i < len(s.currs); i++ {
		curr := s.currs[i]
		if curr == nil {
			continue
		}

		if len(matching) > 0 {
			cmp := curr.IndexInternalID.Compare(matching[0].IndexInternalID)
			if cmp > 0 {
				continue
			}
			if cmp < 0 {
				matching = matching[:0]
				matchingIdxs = matchingIdxs[:0]
			}
		}
		matching = append(matching, curr)
		matchingIdxs = append(matchingIdxs, i)
	}

	s.matching = matching
	s.matchingIdxs = matchingIdxs
}

// score takes the best clause score rather than the sum. It reuses matching[0] as the
// result, like bleve's scorers do, so the caller must not return that one to the pool.
func (s *disMaxSearcher) score(matching []*search.DocumentMatch) *search.DocumentMatch {
	best := matching[0].Score
	for _, docMatch := range matching[1:] {
		if docMatch.Score > best {
			best = docMatch.Score
		}
	}

	var expl *search.Explanation
	if s.explain {
		children := make([]*search.Explanation, len(matching))
		for i, docMatch := range matching {
			children[i] = docMatch.Expl
		}
		expl = &search.Explanation{Value: best, Message: "max of:", Children: children}
	}

	rv := matching[0]
	rv.Score = best
	rv.Expl = expl
	// Locations are built from FieldTermLocations in DocumentMatch.Complete, and they
	// are the only source of highlights. Without this merge only the first clause's
	// language would come back highlighted.
	rv.FieldTermLocations = search.MergeFieldTermLocations(rv.FieldTermLocations, matching[1:])

	return rv
}

func (s *disMaxSearcher) Next(ctx *search.SearchContext) (*search.DocumentMatch, error) {
	if !s.initialized {
		if err := s.initSearchers(ctx); err != nil {
			return nil, err
		}
	}

	var err error
	var rv *search.DocumentMatch

	for rv == nil && len(s.matching) > 0 {
		rv = s.score(s.matching)

		for _, i := range s.matchingIdxs {
			if s.currs[i] != rv {
				ctx.DocumentMatchPool.Put(s.currs[i])
			}
			s.currs[i], err = s.searchers[i].Next(ctx)
			if err != nil {
				return nil, err
			}
		}

		s.updateMatches()
	}

	return rv, nil
}

func (s *disMaxSearcher) Advance(ctx *search.SearchContext,
	ID index.IndexInternalID,
) (*search.DocumentMatch, error) {
	if !s.initialized {
		if err := s.initSearchers(ctx); err != nil {
			return nil, err
		}
	}

	var err error
	for i, searcher := range s.searchers {
		if s.currs[i] != nil {
			if s.currs[i].IndexInternalID.Compare(ID) >= 0 {
				continue
			}
			ctx.DocumentMatchPool.Put(s.currs[i])
		}
		s.currs[i], err = searcher.Advance(ctx, ID)
		if err != nil {
			return nil, err
		}
	}

	s.updateMatches()

	return s.Next(ctx)
}

func (s *disMaxSearcher) Weight() float64 {
	var rv float64
	for _, searcher := range s.searchers {
		rv += searcher.Weight()
	}
	return rv
}

func (s *disMaxSearcher) SetQueryNorm(qnorm float64) {
	for _, searcher := range s.searchers {
		searcher.SetQueryNorm(qnorm)
	}
}

// Count is the worst case, as it is for bleve's own disjunction: a document matching
// several children is counted once per child.
func (s *disMaxSearcher) Count() uint64 {
	var sum uint64
	for _, searcher := range s.searchers {
		sum += searcher.Count()
	}
	return sum
}

func (s *disMaxSearcher) Min() int {
	return 0
}

func (s *disMaxSearcher) Close() (rv error) {
	for _, searcher := range s.searchers {
		if err := searcher.Close(); err != nil && rv == nil {
			rv = err
		}
	}
	return rv
}

func (s *disMaxSearcher) DocumentMatchPoolSize() int {
	rv := len(s.currs)
	for _, searcher := range s.searchers {
		rv += searcher.DocumentMatchPoolSize()
	}
	return rv
}

func (s *disMaxSearcher) Size() int {
	sizeInBytes := reflectStaticSizeDisMaxSearcher + size.SizeOfPtr

	for _, searcher := range s.searchers {
		sizeInBytes += searcher.Size()
	}
	for _, entry := range s.currs {
		if entry != nil {
			sizeInBytes += entry.Size()
		}
	}
	for _, entry := range s.matching {
		if entry != nil {
			sizeInBytes += entry.Size()
		}
	}
	sizeInBytes += len(s.matchingIdxs) * size.SizeOfInt

	return sizeInBytes
}
