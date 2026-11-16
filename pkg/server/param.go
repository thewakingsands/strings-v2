package server

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"xivstrings/pkg/constant"
	"xivstrings/pkg/store"
)

// parseOffsetLimit parses and formats offset and limit from URL query parameters.
// Returns offset (default: 0, min: 0) and limit (default: 100, min: 1, max: 1000).
func parseOffsetLimit(query url.Values) (offset, limit int) {
	offset = 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	limit = 100
	if limitStr := query.Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}

	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	return offset, limit
}

// parseLangs parses the comma separated lang parameter into the languages to search.
// Blank segments are dropped and duplicates are removed, keeping the order of first
// appearance: callers express their language preference through that order.
// An all-blank value yields an empty slice and no error, so the caller can keep
// reporting it as a missing parameter rather than an invalid one.
func parseLangs(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")

	langs := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		lang := strings.TrimSpace(part)
		if lang == "" {
			continue
		}
		if !slices.Contains(constant.Languages, lang) {
			return nil, fmt.Errorf("invalid lang: %s", lang)
		}
		if seen[lang] {
			continue
		}
		seen[lang] = true
		langs = append(langs, lang)
	}

	return langs, nil
}

// parseSheetFilter parses the comma separated sheet parameter of a search. Each entry
// is a glob naming the sheets to search in, or, with a leading -, the sheets to leave
// out. Entries are trimmed and blank ones dropped, so an all-blank value is no filter.
// A - with nothing after it is rejected rather than read as excluding nothing.
func parseSheetFilter(raw string) (store.SheetFilter, error) {
	var filter store.SheetFilter
	for _, part := range strings.Split(raw, ",") {
		entry := strings.TrimSpace(part)
		if entry == "" {
			continue
		}
		if excluded, ok := strings.CutPrefix(entry, "-"); ok {
			excluded = strings.TrimSpace(excluded)
			if excluded == "" {
				return store.SheetFilter{}, fmt.Errorf("invalid sheet: -")
			}
			filter.Exclude = append(filter.Exclude, excluded)
			continue
		}
		filter.Include = append(filter.Include, entry)
	}
	return filter, nil
}

// parseSearchMode reports whether the mode parameter asks for an advanced search.
// Absent means the plain one.
func parseSearchMode(raw string) (advanced bool, err error) {
	switch strings.TrimSpace(raw) {
	case "", "simple":
		return false, nil
	case "advanced":
		return true, nil
	default:
		return false, fmt.Errorf("invalid mode: %s", raw)
	}
}

func parseFields(query url.Values) ([]string, error) {
	fields := query.Get("fields")
	if fields == "" {
		return constant.DefaultDisplayLanguages, nil
	}

	parsedFields := strings.Split(fields, ",")
	for _, field := range parsedFields {
		if !slices.Contains(constant.Languages, field) {
			return nil, fmt.Errorf("invalid field: %s", field)
		}
	}
	return parsedFields, nil
}
