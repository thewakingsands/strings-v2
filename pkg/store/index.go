package store

import (
	"fmt"
	"log"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/lang/cjk"
	"github.com/blevesearch/bleve/v2/analysis/lang/de"
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/analysis/lang/fr"
	"github.com/blevesearch/bleve/v2/mapping"
)

type ItemDocument struct {
	Sheet string `json:"sheet"`
	Id    string `json:"id"`
	Index uint32 `json:"index"`

	// language fields
	Chs string `json:"chs"`
	Tc  string `json:"tc"`
	Ja  string `json:"ja"`
	Ko  string `json:"ko"`
	En  string `json:"en"`
	De  string `json:"de"`
	Fr  string `json:"fr"`
}

func buildItemIndex() mapping.IndexMapping {
	keywordFieldMapping := bleve.NewTextFieldMapping()
	keywordFieldMapping.Analyzer = keyword.Name

	cjkAnalyzer := bleve.NewTextFieldMapping()
	cjkAnalyzer.Analyzer = cjk.AnalyzerName

	enAnalyzer := bleve.NewTextFieldMapping()
	enAnalyzer.Analyzer = en.AnalyzerName

	deAnalyzer := bleve.NewTextFieldMapping()
	deAnalyzer.Analyzer = de.AnalyzerName

	frAnalyzer := bleve.NewTextFieldMapping()
	frAnalyzer.Analyzer = fr.AnalyzerName

	numericFieldMapping := bleve.NewNumericFieldMapping()

	// document
	itemMapping := bleve.NewDocumentMapping()
	itemMapping.AddFieldMappingsAt("id", keywordFieldMapping)
	itemMapping.AddFieldMappingsAt("sheet", keywordFieldMapping)
	itemMapping.AddFieldMappingsAt("index", numericFieldMapping)

	itemMapping.AddFieldMappingsAt("chs", cjkAnalyzer)
	itemMapping.AddFieldMappingsAt("tc", cjkAnalyzer)
	itemMapping.AddFieldMappingsAt("ja", cjkAnalyzer)
	itemMapping.AddFieldMappingsAt("ko", cjkAnalyzer)

	itemMapping.AddFieldMappingsAt("en", enAnalyzer)
	itemMapping.AddFieldMappingsAt("de", deAnalyzer)
	itemMapping.AddFieldMappingsAt("fr", frAnalyzer)

	mapping := bleve.NewIndexMapping()
	mapping.AddDocumentMapping("item", itemMapping)
	mapping.DefaultType = "item"

	return mapping
}

func indexItems(i bleve.Index, items []*Item) error {
	// walk the directory entries for indexing
	startTime := time.Now()
	batch := i.NewBatch()

	for _, item := range items {
		docId := fmt.Sprintf("%s@%s", item.Sheet, item.RowID)
		if err := batch.Index(docId, formatItemDocument(item)); err != nil {
			return err
		}
	}
	// flush the last batch
	err := i.Batch(batch)
	if err != nil {
		return err
	}

	count := len(items)
	indexDuration := time.Since(startTime)
	indexDurationSeconds := float64(indexDuration) / float64(time.Second)
	timePerDoc := float64(indexDuration) / float64(count)
	log.Printf("Indexed %d documents, in %.2fs (average %.2fms/doc)", count, indexDurationSeconds, timePerDoc/float64(time.Millisecond))
	return nil
}
