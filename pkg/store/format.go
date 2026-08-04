package store

import (
	"github.com/blevesearch/bleve/v2/search"
)

func formatItemDocument(item *Item) *ItemDocument {
	return &ItemDocument{
		Sheet: item.Sheet,
		Id:    item.RowID,
		Index: item.Index,
		Chs:   item.Values["chs"],
		Tc:    item.Values["tc"],
		Ja:    item.Values["ja"],
		Ko:    item.Values["ko"],
		En:    item.Values["en"],
		De:    item.Values["de"],
		Fr:    item.Values["fr"],
	}
}

func formatItemFromHit(hit *search.DocumentMatch) *Item {
	values := make(map[string]string)
	var highlights map[string]string

	for key, field := range hit.Fields {
		if key == "sheet" || key == "id" || key == "index" {
			continue
		}

		text, ok := field.(string)
		if !ok {
			continue
		}
		values[key] = text

		// Bleve fragments every highlighted field, even ones the query never matched,
		// and such a fragment is just the head of the value with no <mark> in it. Only
		// fields with term locations carry a real highlight.
		if len(hit.Locations[key]) == 0 {
			continue
		}
		fragments, ok := hit.Fragments[key]
		if !ok || len(fragments) == 0 {
			continue
		}
		if highlights == nil {
			highlights = make(map[string]string)
		}
		highlights[key] = fragments[0]
	}

	return &Item{
		Sheet:      hit.Fields["sheet"].(string),
		RowID:      hit.Fields["id"].(string),
		Values:     values,
		Highlights: highlights,
		Index:      uint32(hit.Fields["index"].(float64)),
	}
}
