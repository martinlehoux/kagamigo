package kauth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/martinlehoux/kagamigo/kcore"
)

func CraftCookie(id kcore.ID, config AuthConfig) http.Cookie {
	expiresAt := time.Now().Add(24 * time.Hour)
	cookieValue := encrypt(config.CookieSecret, fmt.Sprintf("%s:%d", id.String(), expiresAt.Unix()))
	return http.Cookie{
		Domain:  config.Domain,
		Name:    "authentication",
		Value:   cookieValue,
		Expires: expiresAt,
		Path:    "/",
	}
}
