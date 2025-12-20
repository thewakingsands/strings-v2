package server

import (
	"net/url"
	"strconv"
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
