package repository

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
)

func loadDocument[T any, D any](path string, extract func(D) []T, missingDefault func() []T) ([]T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if missingDefault != nil {
				return missingDefault(), nil
			}
			return []T{}, nil
		}
		return nil, err
	}

	var document D
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}

	items := extract(document)
	if items == nil {
		return []T{}, nil
	}
	return items, nil
}

func saveDocument[D any](path string, document D) error {
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}
