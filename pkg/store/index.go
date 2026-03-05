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

func buildItemIndexMapping() mapping.IndexMapping {
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

func buildItemIndex(dataDir string, indexDir string) (bleve.Index, error) {
	log.Printf("Creating new index from %s to %s...", dataDir, indexDir)

	mapping := buildItemIndexMapping()
	idx, err := bleve.New(indexDir, mapping)
	if err != nil {
		return nil, fmt.Errorf("create bleve index: %w", err)
	}

	files, err := deepScanDataFiles(dataDir)
	if err != nil {
		return nil, fmt.Errorf("scan data files: %w", err)
	}

	log.Printf("loading %d files", len(files))

	count := 0
	sheetIndex := make(map[string]uint32)
	for i, path := range files {
		log.Printf("[%d/%d] loading file %s", i+1, len(files), path)
		items, err := loadFile(path)
		if err != nil {
			return nil, fmt.Errorf("load file %s: %w", path, err)
		}

		for _, it := range items {
			// Assign index based on sheet and order in file
			// Index is unique per sheet and follows the order items appear in files
			if _, ok := sheetIndex[it.Sheet]; !ok {
				sheetIndex[it.Sheet] = 0
			}
			it.Index = sheetIndex[it.Sheet]
			sheetIndex[it.Sheet]++
		}

		// Index the item in Bleve
		err = indexItems(idx, items)
		if err != nil {
			return nil, fmt.Errorf("index items: %w", err)
		}

		count += len(items)
	}

	log.Printf("loaded %d items from %d files", count, len(files))

	return idx, nil
}

// BuildIndex builds a new Bleve index from dataDir and writes it to indexDir.
// The index is built and closed so it can be loaded later with LoadStore.
// Use deepScanDataFiles so nested directories (e.g. after extracting zip) are included.
func BuildIndex(dataDir string, indexDir string) error {
	idx, err := buildItemIndex(dataDir, indexDir)
	if err != nil {
		return err
	}
	return idx.Close()
}
