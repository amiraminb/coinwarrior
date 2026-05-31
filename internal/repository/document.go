package repository

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
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

// saveDocument writes document to path atomically and durably: it marshals to a
// uniquely named temp file in the same directory (0600), fsyncs the file, renames
// it over path, then fsyncs the directory so the rename survives a crash.
func saveDocument[D any](path string, document D) error {
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	// Best-effort cleanup; a no-op once the rename succeeds.
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	return syncDir(dir)
}

// syncDir flushes a directory's entries to disk so a rename into it survives a
// crash. Failing to sync a directory is tolerated only when the filesystem does
// not support it (EINVAL/ENOTSUP); all other failures propagate, since a missed
// directory sync silently voids the crash-durability guarantee.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		if errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTSUP) {
			return nil
		}
		return err
	}
	return nil
}
