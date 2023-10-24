package auth

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/martinlehoux/kagamigo/core"
	"golang.org/x/exp/slog"
)

var (
	ErrUserNotLoggedIn = errors.New("user not logged in")
	ErrCookieExpired   = errors.New("cookie expired")
	ErrBadCookie       = errors.New("invalid cookie")
)

type userContext struct{}

func Unauthorized(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusUnauthorized)
	if err != nil {
		_, err = w.Write([]byte(err.Error()))
		core.Expect(err, "error writing response")
	}
}

func CookieAuthMiddleware(loadUser func(context.Context, ...any) (any, error), config AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			cookie, err := r.Cookie("authentication")
			if errors.Is(err, http.ErrNoCookie) {
				next.ServeHTTP(w, r)
				return
			}
			core.Expect(err, "error reading cookie")

			authentication, err := decrypt(config.CookieSecret, cookie.Value)
			if err != nil {
				err = core.Wrap(err, "error decrypting cookie")
				slog.Warn(err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			parts := strings.Split(authentication, ":")
			if len(parts) != 2 {
				slog.Warn(ErrBadCookie.Error())
				http.Error(w, ErrBadCookie.Error(), http.StatusBadRequest)
				return
			}
			userId, err := core.ParseID(parts[0])
			if err != nil {
				err = core.Wrap(err, "error parsing user id")
				slog.Warn(err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			expiresAtSeconds, err := strconv.Atoi(parts[1])
			if err != nil {
				err = core.Wrap(err, "error parsing expires at")
				slog.Warn(err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			expiresAt := time.Unix(int64(expiresAtSeconds), 0)
			if time.Now().After(expiresAt) {
				slog.Warn(ErrCookieExpired.Error())
				http.Error(w, ErrCookieExpired.Error(), http.StatusUnauthorized)
				return
			}
			user, err := loadUser(ctx, userId)
			core.Expect(err, "error loading user")

			ctx = context.WithValue(ctx, userContext{}, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
