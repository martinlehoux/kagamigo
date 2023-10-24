package kcore

import (
	"github.com/kataras/i18n"
)

type Tr = func(format string, args ...any) string

type WithLanguage interface {
	Language() string
}

func GetTr(wl WithLanguage) Tr {
	lang := "en-GB"
	if wl != nil {
		lang = wl.Language()
	}
	return func(format string, args ...any) string { return i18n.Tr(lang, format, args...) }
}

type Login[U any] struct {
	Ok   bool
	User U
	Tr   Tr
}

func LoginFromUser[U WithLanguage](user U, ok bool) Login[U] {
	return Login[U]{
		Ok:   ok,
		User: user,
		Tr:   GetTr(user),
	}
}
