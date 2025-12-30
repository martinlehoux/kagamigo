package ki18n

import (
	"io/fs"

	"github.com/a-h/templ"
	"github.com/kataras/i18n"
)

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

func Tr(lang string, format string, args ...any) templ.Component {
	return templ.Raw(i18n.Tr(lang, format, args...))
}
