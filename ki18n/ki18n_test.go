package ki18n

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/a-h/templ"
	"github.com/stretchr/testify/assert"
)

func TestSimpleText(t *testing.T) {
	initFrenchLocale(map[string]string{
		"Hello you": "Bonjour toi",
	})

	result := render(t, Tr("fr-FR", "Hello you"))
	assert.Equal(t, "Bonjour toi", result)
}

func TestVariables(t *testing.T) {
	initFrenchLocale(map[string]string{
		"Hello %s": "Bonjour %s",
	})

	result := render(t, Tr("fr-FR", "Hello %s", "John"))
	assert.Equal(t, "Bonjour John", result)
}

func TestSpanClass(t *testing.T) {
	initFrenchLocale(map[string]string{
		"Hello <span class=\"text-bold\">%s</span>": "Bonjour <span class=\"text-bold\">%s</span>",
	})

	result := render(t, Tr("fr-FR", "Hello <span class=\"text-bold\">%s</span>", "John"))
	assert.Equal(t, "Bonjour <span class=\"text-bold\">John</span>", result)
}

// render is a test helper that renders a component and returns the result as a string
func render(t *testing.T, c templ.Component) string {
	t.Helper()
	w := &bytes.Buffer{}
	err := c.Render(context.Background(), w)
	assert.NoError(t, err)
	return w.String()
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
