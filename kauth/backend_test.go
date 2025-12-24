package kauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/martinlehoux/kagamigo/kcore"
	"github.com/stretchr/testify/assert"
)

// Test structures
type testUser struct {
	id       kcore.ID
	password string
}

func (u testUser) ID() kcore.ID {
	return u.id
}

type store struct {
	users map[string]testUser
}

func (s *store) Authenticate(ctx context.Context, username, password string) (testUser, bool) {
	if u, ok := s.users[username]; ok {
		if password == u.password {
			return u, true
		}
	}
	return testUser{}, false
}

func (s *store) LoadUser(ctx context.Context, id kcore.ID) (testUser, error) {
	for _, u := range s.users {
		if u.id == id {
			return u, nil
		}
	}
	return testUser{}, errors.New("user not found")
}

func newUser(password string) testUser {
	return testUser{id: kcore.NewID(), password: password}
}

func newBackend() (*Backend[testUser], *store) {
	s := store{users: map[string]testUser{}}
	return &Backend[testUser]{
		UserStore:    &s,
		Domain:       "localhost",
		CookieSecret: GenerateCookieSecret(),
		Now:          time.Now,
	}, &s
}

// Helper to extract authentication cookie value from a response recorder
func getAuthCookie(rr *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == "authentication" {
			return c
		}
	}
	return nil
}

// Test Backend.Login sets cookie on success
func TestBackend_Login_Success(t *testing.T) {
	user := newUser("pass")
	backend, store := newBackend()
	store.users["user"] = user

	rr := httptest.NewRecorder()
	ctx := context.Background()
	_, err := backend.Login(rr, ctx, "user", "pass")
	assert.NoError(t, err)

	c := getAuthCookie(rr)
	assert.NotNil(t, c, "authentication cookie should be set")
	assert.Equal(t, backend.Domain, c.Domain)
	assert.Equal(t, "/", c.Path)
	assert.True(t, time.Until(c.Expires) >= 23*time.Hour && time.Until(c.Expires) <= 25*time.Hour, "expected expires about 24h from now")
}

// Test Backend.Login returns error on invalid credentials
func TestBackend_Login_InvalidCredentials(t *testing.T) {
	user := newUser("pass")
	backend, store := newBackend()
	store.users["user"] = user

	rr := httptest.NewRecorder()
	ctx := context.Background()
	_, err := backend.Login(rr, ctx, "baduser", "badpass")
	assert.Equal(t, ErrInvalidCredentials, err)

	c := getAuthCookie(rr)
	assert.Nil(t, c, "did not expect authentication cookie to be set")
}

// Test Backend.Logout deletes the authentication cookie
func TestBackend_Logout(t *testing.T) {
	user := newUser("pass")
	backend, store := newBackend()
	store.users["user"] = user

	rr := httptest.NewRecorder()
	backend.Logout(rr)

	c := getAuthCookie(rr)
	assert.NotNil(t, c, "authentication cookie should be set for deletion")
	assert.Equal(t, "", c.Value)
	assert.Equal(t, "/", c.Path)
	assert.Equal(t, backend.Domain, c.Domain)
	assert.Equal(t, int(-1), c.MaxAge)
	assert.True(t, c.Expires.Equal(time.Unix(0, 0)), "Expires should be Unix epoch")
	assert.True(t, c.HttpOnly)
	assert.True(t, c.Secure)
}

func TestCookieAuthMiddleware_ValidCookie_SetsUserInContext(t *testing.T) {
	user := newUser("pass")
	backend, store := newBackend()
	store.users["user"] = user

	rrLogin := httptest.NewRecorder()
	ctx := context.Background()
	_, _ = backend.Login(rrLogin, ctx, "user", "pass")
	cookie := getAuthCookie(rrLogin)
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	var userInHandler testUser
	handler := backend.CookieAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := backend.User(r.Context())
		assert.True(t, ok, "user should be in context")
		userInHandler = u
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, user.id, userInHandler.id)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCookieAuthMiddleware_NoCookie_PassesThrough(t *testing.T) {
	user := newUser("pass")
	backend, store := newBackend()
	store.users["user"] = user
	req := httptest.NewRequest("GET", "/", nil)

	called := false
	handler := backend.CookieAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.True(t, called, "handler should be called")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCookieAuthMiddleware_InvalidCookie_DecryptionFails(t *testing.T) {
	user := newUser("pass")
	backend, store := newBackend()
	store.users["user"] = user
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "authentication", Value: "not-encrypted"})

	handler := backend.CookieAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called on decryption failure")
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCookieAuthMiddleware_BadFormatCookie(t *testing.T) {
	user := newUser("pass")
	backend, store := newBackend()
	store.users["user"] = user
	// Encrypt a string that doesn't have the expected "id:expires" format
	badValue := encrypt(backend.CookieSecret, "badformat")
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "authentication", Value: badValue})

	handler := backend.CookieAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called on bad format")
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

func TestCookieAuthMiddleware_ExpiredCookie(t *testing.T) {
	user := newUser("pass")
	backend, store := newBackend()
	store.users["user"] = user

	backend.Now = func() time.Time { return Must(time.Parse(time.RFC3339, "2025-12-23T12:00:00Z")) }
	rrLogin := httptest.NewRecorder()
	ctx := context.Background()
	_, _ = backend.Login(rrLogin, ctx, "user", "pass")
	cookie := getAuthCookie(rrLogin)
	assert.NotNil(t, cookie, "authentication cookie should be set")
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	var userSet bool
	backend.Now = func() time.Time { return Must(time.Parse(time.RFC3339, "2025-12-24T13:00:00Z")) } // after expiry
	handler := backend.CookieAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := backend.User(r.Context())
		userSet = ok
		w.WriteHeader(http.StatusUnauthorized)
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.False(t, userSet, "user should not be set in context for expired cookie")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestCookieAuthMiddleware_InvalidUserID(t *testing.T) {
	user := newUser("pass")
	backend, store := newBackend()
	store.users["user"] = user
	rrLogin := httptest.NewRecorder()
	ctx := context.Background()
	_, _ = backend.Login(rrLogin, ctx, "user", "pass")
	cookie := getAuthCookie(rrLogin)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)
	delete(store.users, "user") // User is no longer valid

	handler := backend.CookieAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called on invalid user id")
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
