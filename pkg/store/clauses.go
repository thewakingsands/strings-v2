package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"

	"github.com/blevesearch/bleve/v2/search/searcher"
)

// maxClauseCount caps the terms one wildcard, regexp or fuzzy clause may expand to, and
// the alternatives one disjunction may hold.
//
// Bleve expands a clause before it first looks at the context, so without a cap no
// deadline can stop a CJK term with fuzziness 2, which matches most of a column's
// dictionary, from holding a core for tens of seconds. This value clears every sheet
// name (under 7,000), a one-letter prefix in en, and the longest value in the corpus
// pasted whole (under 2,000 terms).
const maxClauseCount = 8192

func init() {
	searcher.DisjunctionMaxClauseCount = maxClauseCount
}

// QueryTooBroadError is a query that expands to more than maxClauseCount terms.
type QueryTooBroadError struct {
	field string // empty when bleve counted the alternatives of a disjunction instead
	limit int
	err   error
}

func (e *QueryTooBroadError) Error() string {
	if e.field == "" {
		return fmt.Sprintf("query combines more than %d terms; shorten it", e.limit)
	}
	return fmt.Sprintf("query expands to more than %d terms in %s; "+
		"make the wildcard, regular expression or fuzzy clause more specific", e.limit, e.field)
}

func (e *QueryTooBroadError) Unwrap() error { return e.err }

// Bleve reports the clause limit as formatted text only, with no error value to match.
var tooManyClauses = regexp.MustCompile("^TooManyClauses over field: `([^`]*)`")

// searchFailed returns what a search reports when bleve fails with err.
func searchFailed(err error) error {
	if m := tooManyClauses.FindStringSubmatch(err.Error()); m != nil {
		return &QueryTooBroadError{field: m[1], limit: searcher.DisjunctionMaxClauseCount, err: err}
	}
	// The UI cancels the search it replaces, so a client going away is routine.
	if !errors.Is(err, context.Canceled) {
		log.Printf("search error: %v", err)
	}
	return fmt.Errorf("search error: %w", err)
}
