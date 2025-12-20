package store

// Item represents one string entry exported from ixion (per sheet/row).
type Item struct {
	Sheet  string            `json:"sheet"`
	RowID  string            `json:"rowId"`
	Values map[string]string `json:"values"`
	Index  uint32            `json:"index"`
}
