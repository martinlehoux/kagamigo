package kauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/martinlehoux/kagamigo/kcore"
	"golang.org/x/exp/slog"
)

var (
	ErrUserNotLoggedIn    = errors.New("user not logged in")
	ErrCookieExpired      = errors.New("cookie expired")
	ErrBadCookie          = errors.New("invalid cookie")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type identifyable interface {
	ID() kcore.ID
}

// A backend needs a user store to authenticate and load users
type UserStore[U identifyable] interface {
	LoadUser(ctx context.Context, id kcore.ID) (U, error)
	Authenticate(ctx context.Context, username string, password string) (U, bool)
}

// A backend is responsible for authenticating users and managing their sessions
type Backend[U identifyable] struct {
	UserStore[U]
	Domain       string
	CookieSecret []byte
	Now          func() time.Time // injectable time provider
}

// Authenticate and persist the login in the user session
func (b *Backend[U]) Login(w http.ResponseWriter, ctx context.Context, username, password string) (U, error) {
	user, ok := b.Authenticate(ctx, username, password)
	if !ok {
		return user, ErrInvalidCredentials
	}
	now := time.Now
	if b.Now != nil {
		now = b.Now
	}
	expiresAt := now().Add(24 * time.Hour)
	cookie := http.Cookie{
		Domain:  b.Domain,
		Name:    "authentication",
		Value:   encrypt(b.CookieSecret, fmt.Sprintf("%s:%d", user.ID().String(), expiresAt.Unix())),
		Expires: expiresAt,
		Path:    "/",
	}
	http.SetCookie(w, &cookie)
	return user, nil
}

// Clears login from the response
func (b *Backend[U]) Logout(w http.ResponseWriter) {
	// Set the cookie with MaxAge -1 to delete it
	http.SetCookie(w, &http.Cookie{
		Name:     "authentication",
		Value:    "",
		Path:     "/",
		Domain:   b.Domain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
	})
}

type userContext struct{}

func (b *Backend[U]) PersistUser(r *http.Request, user U) *http.Request {
	ctx := context.WithValue(r.Context(), userContext{}, user)
	return r.WithContext(ctx)
}

func (b Backend[U]) CookieAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			cookie, err := r.Cookie("authentication")
			if errors.Is(err, http.ErrNoCookie) {
				next.ServeHTTP(w, r)
				return
			}
			kcore.Expect(err, "error reading cookie")

			authentication, err := decrypt(b.CookieSecret, cookie.Value)
			if err != nil {
				err = kcore.Wrap(err, "error decrypting cookie")
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
			userId, err := kcore.ParseID(parts[0])
			if err != nil {
				err = kcore.Wrap(err, "error parsing user id")
				slog.Warn(err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			expiresAtSeconds, err := strconv.Atoi(parts[1])
			if err != nil {
				err = kcore.Wrap(err, "error parsing expires at")
				slog.Warn(err.Error())
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			now := time.Now
			if b.Now != nil {
				now = b.Now
			}
			expiresAt := time.Unix(int64(expiresAtSeconds), 0)
			if now().After(expiresAt) {
				slog.Warn(ErrCookieExpired.Error())
				http.Error(w, ErrCookieExpired.Error(), http.StatusUnauthorized)
				return
			}
			user, err := b.LoadUser(ctx, userId)
			if err != nil {
				err = kcore.Wrap(err, "error loading user")
				slog.Warn(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			r = b.PersistUser(r, user)

			ctx = context.WithValue(ctx, userContext{}, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (b *Backend[U]) User(ctx context.Context) (U, bool) {
	user, ok := ctx.Value(userContext{}).(U)
	return user, ok
}
