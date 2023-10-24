package core

import (
	"bytes"
	"context"
	"html/template"
	"net/http"

	"github.com/a-h/templ"
)

func ExecuteTemplate(w http.ResponseWriter, tpl template.Template, name string, data interface{}) {
	var buf bytes.Buffer
	err := tpl.ExecuteTemplate(&buf, name, data)
	Expect(err, "error executing template")
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	_, err = buf.WriteTo(w)
	Expect(err, "error writing template to response writer")
}

func RenderPage(ctx context.Context, page templ.Component, w http.ResponseWriter) {
	var buf bytes.Buffer
	err := page.Render(ctx, &buf)
	Expect(err, "error rendering page")
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	_, err = buf.WriteTo(w)
	Expect(err, "error writing page to response writer")
}
