package kstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type document struct {
	Name  string `json:"name" yaml:"name"`
	Count int    `json:"count" yaml:"count"`
}

func TestJSONRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "document.json")
	want := document{Name: "example", Count: 3}

	require.NoError(t, Save(path, want, JSON, 0o600))
	got, err := Load[document](path, JSON)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestYAMLRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "document.yaml")
	want := document{Name: "example", Count: 3}

	require.NoError(t, Save(path, want, YAML, 0o640))
	got, err := Load[document](path, YAML)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestLoadMalformedDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "document.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

	_, err := Load[document](path, JSON)

	assert.Error(t, err)
}

func TestLoadMissingDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")

	_, err := Load[document](path, JSON)

	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestSaveSetsRequestedPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "document.json")

	require.NoError(t, Save(path, document{}, JSON, 0o640))
	info, err := os.Stat(path)

	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o640), info.Mode().Perm())
}

func TestSaveFailedWriteLeavesExistingDocumentUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "document.json")
	require.NoError(t, Save(path, document{Name: "original"}, JSON, 0o600))

	err := Save(filepath.Join(path, "child"), document{Name: "replacement"}, JSON, 0o600)

	assert.Error(t, err)
	loaded, loadErr := Load[document](path, JSON)
	require.NoError(t, loadErr)
	assert.Equal(t, "original", loaded.Name)
}

func TestSaveReturnsCodecErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "document.json")
	codec := failingCodec{err: errors.New("codec failed")}

	err := Save(path, document{}, codec, 0o600)

	assert.ErrorIs(t, err, codec.err)
}

type failingCodec struct{ err error }

func (c failingCodec) Marshal(any) ([]byte, error) { return nil, c.err }

func (c failingCodec) Unmarshal([]byte, any) error { return c.err }
