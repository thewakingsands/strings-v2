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
	for key, field := range hit.Fields {
		if key == "sheet" || key == "id" || key == "index" {
			continue
		}

		if fragments, ok := hit.Fragments[key]; ok {
			values[key] = fragments[0]
		} else {
			values[key] = field.(string)
		}
	}

	return &Item{
		Sheet:  hit.Fields["sheet"].(string),
		RowID:  hit.Fields["id"].(string),
		Values: values,
		Index:  uint32(hit.Fields["index"].(float64)),
	}
}
