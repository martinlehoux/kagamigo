package ki18n

import (
	"bytes"
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestSimpleText(t *testing.T) {
	fs := fstest.MapFS{
		"fr-FR/index.yml": &fstest.MapFile{
			Data: []byte("\"Hello you\": \"Bonjour toi\""),
		},
	}
	Init(fs)
	c := Tr("fr-FR", "Hello you")
	w := &bytes.Buffer{}
	err := c.Render(context.Background(), w)
	assert.NoError(t, err)
	assert.Equal(t, "Bonjour toi", w.String())
}

func TestVariables(t *testing.T) {
	fs := fstest.MapFS{
		"fr-FR/index.yml": &fstest.MapFile{
			Data: []byte("\"Hello %s\": \"Bonjour %s\""),
		},
	}
	Init(fs)
	c := Tr("fr-FR", "Hello %s", "John")
	w := &bytes.Buffer{}
	err := c.Render(context.Background(), w)
	assert.NoError(t, err)
	assert.Equal(t, "Bonjour John", w.String())
}
