package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitRecordExtractor_Extract(t *testing.T) {
	dir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}

	run("git", "init")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "Test")

	content := "line one\n// TODO: fix this\nline three\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte(content), 0600))
	run("git", "add", "file.txt")
	run("git", "commit", "-m", "initial")

	factory, err := NewGitRecordExtractorFactory(dir)
	require.NoError(t, err)

	extractor, err := factory.CreateExtractor("file.txt")
	require.NoError(t, err)

	record, err := extractor.Extract("TODO", 2)
	require.NoError(t, err)

	assert.Equal(t, 2, record.line)
	assert.Equal(t, "TODO", record.keyword)
	assert.False(t, record.date.IsZero())
}

func TestGitRecordExtractor_UntrackedFile(t *testing.T) {
	dir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}

	run("git", "init")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "Test")
	run("git", "commit", "--allow-empty", "-m", "initial")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("TODO\n"), 0600))

	factory, err := NewGitRecordExtractorFactory(dir)
	require.NoError(t, err)

	_, err = factory.CreateExtractor("untracked.txt")
	assert.ErrorIs(t, err, ErrNotTracked)
}
