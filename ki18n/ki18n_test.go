package ki18n

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/a-h/templ"
	"github.com/martinlehoux/kagamigo/kcore"
	"github.com/stretchr/testify/assert"
)

func TestSimpleText(t *testing.T) {
	initFrenchLocale(map[string]string{
		"Hello you": "Bonjour toi",
	})
	ctx := context.WithValue(context.Background(), contextKey{}, "fr-FR")

	result := render(t, Tr(ctx, "Hello you"))
	assert.Equal(t, "Bonjour toi", result)
}

func TestVariables(t *testing.T) {
	initFrenchLocale(map[string]string{
		"Hello %s": "Bonjour %s",
	})
	ctx := context.WithValue(context.Background(), contextKey{}, "fr-FR")

	result := render(t, Tr(ctx, "Hello %s", "John"))
	assert.Equal(t, "Bonjour John", result)
}

func TestSpanClass(t *testing.T) {
	initFrenchLocale(map[string]string{
		"Hello <span class=\"text-bold\">%s</span>": "Bonjour <span class=\"text-bold\">%s</span>",
	})
	ctx := context.WithValue(context.Background(), contextKey{}, "fr-FR")

	result := render(t, Tr(ctx, "Hello <span class=\"text-bold\">%s</span>", "John"))
	assert.Equal(t, "Bonjour <span class=\"text-bold\">John</span>", result)
}

func TestTr_withoutLangInContext(t *testing.T) {
	initFrenchLocale(map[string]string{
		"Hello": "Bonjour",
	})

	result := render(t, Tr(context.Background(), "Hello"))
	assert.Equal(t, "Hello", result)
}

func TestCookieStrategy_found(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "lang", Value: "fr-FR"})

	s := CookieStrategy("lang")
	assert.Equal(t, "fr-FR", s.Detect(r))
}

func TestCookieStrategy_missing(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	s := CookieStrategy("lang")
	assert.Equal(t, "", s.Detect(r))
}

func TestAcceptLanguageStrategy_found(t *testing.T) {
	initFrenchLocale(map[string]string{})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")

	assert.Equal(t, "fr-FR", AcceptLanguageStrategy.Detect(r))
}

func TestAcceptLanguageStrategy_missing(t *testing.T) {
	initFrenchLocale(map[string]string{})
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	assert.Equal(t, "", AcceptLanguageStrategy.Detect(r))
}

func TestLangMiddleware_firstStrategyWins(t *testing.T) {
	first := strategyFunc(func(_ *http.Request) string { return "fr-FR" })
	second := strategyFunc(func(_ *http.Request) string { return "de-DE" })

	var capturedLang string
	handler := LangMiddleware(first, second)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedLang, _ = r.Context().Value(contextKey{}).(string)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, "fr-FR", capturedLang)
}

func TestLangMiddleware_fallsThrough(t *testing.T) {
	first := strategyFunc(func(_ *http.Request) string { return "" })
	second := strategyFunc(func(_ *http.Request) string { return "fr-FR" })

	var capturedLang string
	handler := LangMiddleware(first, second)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedLang, _ = r.Context().Value(contextKey{}).(string)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, "fr-FR", capturedLang)
}

func TestLangMiddleware_fallback(t *testing.T) {
	first := strategyFunc(func(_ *http.Request) string { return "" })

	var capturedLang string
	handler := LangMiddleware(first)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedLang, _ = r.Context().Value(contextKey{}).(string)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, "en-GB", capturedLang)
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
	kcore.Expect(Init(fs), "failed to init fs")
}

// strategyFunc is a test helper to create a Strategy from a plain function
type strategyFunc func(*http.Request) string

func (f strategyFunc) Detect(r *http.Request) string { return f(r) }
