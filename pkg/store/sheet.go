package store

import (
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
)

// SheetFilter restricts a search to some sheets. The zero value means no filter.
//
// Every entry is a case-sensitive glob over the sheet name, where * matches any run of
// characters (/ included) and ? exactly one; an entry without either is an exact name.
// A row passes when it matches at least one of Include, or Include is empty, and none
// of Exclude.
type SheetFilter struct {
	Include []string
	Exclude []string
}

// apply restricts text to the sheets f lets through, leaving the order of the rows it
// keeps exactly as text alone would rank them.
func (f SheetFilter) apply(text query.Query) query.Query {
	if len(f.Include) == 0 && len(f.Exclude) == 0 {
		return text
	}

	// One exact sheet is the common case, and a conjunction leapfrogs through it, where
	// a filter has to visit every text hit: an order of magnitude faster on a broad
	// query. With a single term the conjunction's extra score is the same for every
	// row, so the order is unchanged.
	if len(f.Include) == 1 && len(f.Exclude) == 0 && !isSheetPattern(f.Include[0]) {
		return bleve.NewConjunctionQuery(text, sheetPatternQuery(f.Include[0]))
	}

	// Several sheet terms cannot go into a scored conjunction: each brings its own IDF,
	// which lifts rows from rare sheets. Boosting the clause to zero does not help
	// either, since it still changes the query norm. Filter and MustNot never score.
	boolean := bleve.NewBooleanQuery()
	boolean.AddMust(text)
	if len(f.Include) > 0 {
		boolean.AddFilter(sheetPatternsQuery(f.Include))
	}
	if len(f.Exclude) > 0 {
		boolean.AddMustNot(sheetPatternsQuery(f.Exclude))
	}
	return boolean
}

// sheetPatternsQuery matches a sheet matching any of patterns.
func sheetPatternsQuery(patterns []string) query.Query {
	if len(patterns) == 1 {
		return sheetPatternQuery(patterns[0])
	}
	disjuncts := make([]query.Query, 0, len(patterns))
	for _, p := range patterns {
		disjuncts = append(disjuncts, sheetPatternQuery(p))
	}
	return bleve.NewDisjunctionQuery(disjuncts...)
}

// sheetPatternQuery matches a sheet named by the glob p.
func sheetPatternQuery(p string) query.Query {
	if !isSheetPattern(p) {
		term := bleve.NewTermQuery(p)
		term.SetField("sheet")
		return term
	}
	wildcard := bleve.NewWildcardQuery(p)
	wildcard.SetField("sheet")
	return wildcard
}

func isSheetPattern(p string) bool {
	return strings.ContainsAny(p, "*?")
}
