package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func scanDataFiles(dataDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".json") {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func loadFile(path string) ([]*Item, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	var arr []*Item
	if err := decoder.Decode(&arr); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	return arr, nil
}
