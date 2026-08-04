package store

// Item represents one string entry exported from ixion (per sheet/row).
//
// Values holds the complete, raw field values. Highlights holds the matching
// snippet for each language that the search actually hit: HTML escaped, with the
// matches wrapped in <mark>, and shortened with an ellipsis when the value is long.
// It is absent for lookups that do not search, such as GetBySheet.
type Item struct {
	Sheet      string            `json:"sheet"`
	RowID      string            `json:"rowId"`
	Values     map[string]string `json:"values"`
	Highlights map[string]string `json:"highlights,omitempty"`
	Index      uint32            `json:"index"`
}
