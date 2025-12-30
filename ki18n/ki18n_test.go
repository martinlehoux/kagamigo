package ki18n

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestSimpleText(t *testing.T) {
	initFrenchLocale(map[string]string{
		"Hello you": "Bonjour toi",
	})

	c := Tr("fr-FR", "Hello you")
	w := &bytes.Buffer{}
	err := c.Render(context.Background(), w)
	assert.NoError(t, err)
	assert.Equal(t, "Bonjour toi", w.String())
}

func TestVariables(t *testing.T) {
	initFrenchLocale(map[string]string{
		"Hello %s": "Bonjour %s",
	})

	c := Tr("fr-FR", "Hello %s", "John")
	w := &bytes.Buffer{}
	err := c.Render(context.Background(), w)
	assert.NoError(t, err)
	assert.Equal(t, "Bonjour John", w.String())
}

// initFrenchLocale creates a filesystem with fr-FR locale from a translation map
func initFrenchLocale(translations map[string]string) {
	var sb strings.Builder
	for key, value := range translations {
		fmt.Fprintf(&sb, "%q: %q\n", key, value)
	}

	fs := fstest.MapFS{
		"fr-FR/index.yml": &fstest.MapFile{
			Data: []byte(sb.String()),
		},
	}
	Init(fs)
}
