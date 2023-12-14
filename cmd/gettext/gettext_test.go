package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractKeys(t *testing.T) {
	content := `{ login.Tr("Hello") }`

	keys := extractKeys(content)

	assert.Len(t, keys, 1)
	assert.Equal(t, 0, keys["Hello"])
}

func TestExtractKeysWithoutSpaces(t *testing.T) {
	content := `{login.Tr("Hello")}`

	keys := extractKeys(content)

	assert.Len(t, keys, 1)
	assert.Equal(t, 0, keys["Hello"])
}

func TestExtractKeysWithOneArgs(t *testing.T) {
	content := `{ login.Tr("approveButton", a.Username) }`

	keys := extractKeys(content)

	assert.Len(t, keys, 1)
	assert.Equal(t, 1, keys["approveButton"])
}

func TestExtractKeysWithSeveralArgs(t *testing.T) {
	content := `{ login.Tr("approveButton", a.Username, a.Age, a.Email) }`

	keys := extractKeys(content)

	assert.Len(t, keys, 1)
	assert.Equal(t, 3, keys["approveButton"])
}

func TestExtractKeysWithComplexArgs(t *testing.T) {
	content := `{ login.Tr("raceStart_chosen", a.StartAt.Format("Monday, January 2, 2006 at 15:04")) }`

	keys := extractKeys(content)

	assert.Len(t, keys, 1)
	assert.Equal(t, 1, keys["raceStart_chosen"])
}

func TestExtractMultipleKeys(t *testing.T) {
	content := `{ login.Tr("test_1", a.StartAt.Format("Monday, January 2, 2006 at 15:04")) }{ login.Tr("test_2", a.Test) }`

	keys := extractKeys(content)

	assert.Len(t, keys, 2)
	assert.Equal(t, 1, keys["test_1"])
	assert.Equal(t, 1, keys["test_2"])
}

func TestExtractKeyFromSimpleTrFunc(t *testing.T) {
	content := `{ Tr("test") }`

	keys := extractKeys(content)

	assert.Len(t, keys, 1)
	assert.Equal(t, 0, keys["test"])
}
