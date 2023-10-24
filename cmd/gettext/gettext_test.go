package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractKeys(t *testing.T) {
	content := `{{ call .T "Hello" }}`

	keys := extractKeys(content)

	assert.Len(t, keys, 1)
	assert.Equal(t, 0, keys["Hello"])
}

func TestExtractKeysWithoutSpaces(t *testing.T) {
	content := `{{call .T "Hello"}}`

	keys := extractKeys(content)

	assert.Len(t, keys, 1)
	assert.Equal(t, 0, keys["Hello"])
}

func TestExtractKeysWithDollar(t *testing.T) {
	content := `{{ call $.T "Hello" }}`

	keys := extractKeys(content)

	assert.Len(t, keys, 1)
	assert.Equal(t, 0, keys["Hello"])
}

func TestExtractKeysWithOneArgs(t *testing.T) {
	content := `{{ call .T "approveButton" .Username }}`

	keys := extractKeys(content)

	assert.Len(t, keys, 1)
	assert.Equal(t, 1, keys["approveButton"])
}

func TestExtractKeysWithSeveralArgs(t *testing.T) {
	content := `{{ call .T "approveButton" .Username .Age .Email }}`

	keys := extractKeys(content)

	assert.Len(t, keys, 1)
	assert.Equal(t, 3, keys["approveButton"])
}

func TestExtractKeysWithComplexArgs(t *testing.T) {
	content := `{{ call $.T "raceStart_chosen" (.StartAt.Format "Monday, January 2, 2006 at 15:04") }}`

	keys := extractKeys(content)

	assert.Len(t, keys, 1)
	assert.Equal(t, 1, keys["raceStart_chosen"])
}
