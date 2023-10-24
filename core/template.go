package core

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

type LoginContext[U any] struct {
	IsLoggedIn bool
	User       U
	Tr         Tr
}

func GetLoginContext[U WithLanguage](user U, ok bool) LoginContext[U] {
	return LoginContext[U]{
		IsLoggedIn: ok,
		User:       user,
		Tr:         GetTr(user),
	}
}
