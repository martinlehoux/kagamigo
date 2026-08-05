// Package kstore provides durable, typed document files.
package kstore

import (
	"fmt"
	"os"
	"path/filepath"
)

// Codec serializes and deserializes document values.
type Codec interface {
	Marshal(value any) ([]byte, error)
	Unmarshal(data []byte, value any) error
}

// Load reads and decodes a document from path.
func Load[T any](path string, codec Codec) (T, error) {
	var value T

	data, err := os.ReadFile(path) // #nosec G304 -- Callers explicitly provide the document path.
	if err != nil {
		return value, fmt.Errorf("read document %q: %w", path, err)
	}
	if err := codec.Unmarshal(data, &value); err != nil {
		return value, fmt.Errorf("decode document %q: %w", path, err)
	}

	return value, nil
}

// Save encodes value and atomically replaces path with the requested mode.
func Save(path string, value any, codec Codec, mode os.FileMode) error {
	data, err := codec.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode document %q: %w", path, err)
	}

	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary document for %q: %w", path, err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath) //nolint:errcheck // Best-effort cleanup after close or rename.

	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set permissions on temporary document for %q: %w", path, err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary document for %q: %w", path, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary document for %q: %w", path, err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace document %q: %w", path, err)
	}

	return nil
}
