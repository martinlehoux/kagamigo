package auth

import (
	"encoding/hex"
	"errors"
	"os"

	"github.com/martinlehoux/kagamigo/core"
	"golang.org/x/exp/slog"
)

var (
	ErrCookieBadLength = errors.New("cookie secret must be 32 bytes")
)

type AuthConfig struct {
	Domain       string
	CookieSecret []byte
}

func LoadCookieSecret(cookieSecretString string) []byte {
	cookiesSecret, err := hex.DecodeString(cookieSecretString)
	if err != nil {
		err = core.Wrap(err, "error decoding cookie secret")
		slog.Error(err.Error())
		os.Exit(1)
	}
	if len(cookiesSecret) != 32 {
		slog.Error(ErrCookieBadLength.Error())
		os.Exit(1)
	}
	slog.Info("cookie secret loaded")
	return cookiesSecret
}
