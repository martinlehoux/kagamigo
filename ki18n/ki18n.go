package ki18n

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/kataras/i18n"
)

type contextKey struct{}

const defaultLang = "en-GB"

type Strategy interface {
	Detect(r *http.Request) string
}

type cookieStrategy struct{ name string }

func (s cookieStrategy) Detect(r *http.Request) string {
	c, err := r.Cookie(s.name)
	if err != nil {
		return ""
	}
	return c.Value
}

func CookieStrategy(name string) Strategy {
	return cookieStrategy{name: name}
}

type acceptLanguageStrategy struct{}

func (acceptLanguageStrategy) Detect(r *http.Request) string {
	lang := i18n.Default.GetLocale(r).Language()
	if lang == defaultLang {
		if r.Header.Get("Accept-Language") == "" {
			return ""
		}
	}
	return lang
}

var AcceptLanguageStrategy Strategy = acceptLanguageStrategy{}

func LangMiddleware(strategies ...Strategy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lang := ""
			for _, s := range strategies {
				if l := s.Detect(r); l != "" {
					lang = l
					break
				}
			}
			if lang == "" {
				lang = defaultLang
			}
			ctx := context.WithValue(r.Context(), contextKey{}, lang)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

var frenchMonths = [...]string{
	"janvier", "février", "mars", "avril", "mai", "juin",
	"juillet", "août", "septembre", "octobre", "novembre", "décembre",
}

// Locale configures a language for use with ki18n. Lang is a BCP-47 tag (e.g. "es-ES").
// FormatTime is optional; if nil, en-GB formatting is used.
type Locale struct {
	Lang       string
	FormatTime func(time.Time) string
}

var defaultLocales = []Locale{
	{
		Lang:       "en-GB",
		FormatTime: func(t time.Time) string { return t.Format("2 January 2006") },
	},
	{
		Lang: "fr-FR",
		FormatTime: func(t time.Time) string {
			return fmt.Sprintf("%d %s %d", t.Day(), frenchMonths[t.Month()-1], t.Year())
		},
	},
}

var timeFormatters map[string]func(time.Time) string

func FormatTime(ctx context.Context, t time.Time) string {
	lang, _ := ctx.Value(contextKey{}).(string)
	f, ok := timeFormatters[lang]
	if !ok {
		f = timeFormatters[defaultLang]
	}
	return f(t)
}

func Tr(ctx context.Context, format string, args ...any) templ.Component {
	lang, _ := ctx.Value(contextKey{}).(string)
	if lang == "" {
		lang = defaultLang
	}
	msg := i18n.Tr(lang, format, args...)
	if msg == "" {
		msg = format
	}
	return templ.Raw(msg)
}

func Init(localesFS fs.FS, extra ...Locale) error {
	all := append(defaultLocales, extra...)

	langs := make([]string, len(all))
	timeFormatters = make(map[string]func(time.Time) string, len(all))
	for i, loc := range all {
		langs[i] = loc.Lang
		if loc.FormatTime != nil {
			timeFormatters[loc.Lang] = loc.FormatTime
		} else {
			timeFormatters[loc.Lang] = timeFormatters[defaultLang]
		}
	}

	loader, err := i18n.FS(localesFS, "*/*.yml")
	if err != nil {
		return err
	}
	inst, err := i18n.New(loader, langs...)
	if err != nil {
		return err
	}
	i18n.Default = inst
	return nil
}
