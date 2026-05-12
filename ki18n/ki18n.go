package ki18n

import (
	"context"
	"io/fs"
	"net/http"

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

func Init(fs fs.FS) error {
	loader, err := i18n.FS(fs, "*/*.yml")
	if err != nil {
		return err
	}
	inst, err := i18n.New(loader, "en-GB", "fr-FR")
	if err != nil {
		return err
	}
	i18n.Default = inst
	return nil
}
