package kauth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"

	"github.com/martinlehoux/kagamigo/kcore"
	"golang.org/x/exp/slog"
)

var (
	ErrEncryptedTooShort = errors.New("encrypted text is too short")
	ErrCookieBadLength   = errors.New("cookie secret must be 32 bytes")
)

// Generate a random 32-byte secret for cookies
func GenerateCookieSecret() []byte {
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	kcore.Expect(err, "failed to generate random bytes")
	return secret
}

func LoadCookieSecret(cookieSecretString string) []byte {
	cookiesSecret, err := hex.DecodeString(cookieSecretString)
	if err != nil {
		err = kcore.Wrap(err, "error decoding cookie secret")
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

func encrypt(secret []byte, plainText string) string {
	block, err := aes.NewCipher(secret)
	kcore.Expect(err, "error creating AES cipher")

	aead, err := cipher.NewGCM(block)
	kcore.Expect(err, "error creating AEAD")

	nonce := make([]byte, aead.NonceSize())
	_, err = rand.Read(nonce)
	kcore.Expect(err, "error generating nonce")

	return base64.URLEncoding.EncodeToString(aead.Seal(nonce, nonce, []byte(plainText), nil))
}

func decrypt(secret []byte, encryptedText string) (string, error) {
	encryptedBytes, err := base64.URLEncoding.DecodeString(encryptedText)
	if err != nil {
		err = kcore.Wrap(err, "error decoding base64")
		return "", err
	}
	block, err := aes.NewCipher(secret)
	kcore.Expect(err, "error creating AES cipher")

	aead, err := cipher.NewGCM(block)
	kcore.Expect(err, "error creating AEAD")

	nonceSize := aead.NonceSize()
	if len(encryptedBytes) < nonceSize {
		return "", ErrEncryptedTooShort
	}
	nonce, cipherText := encryptedBytes[:nonceSize], encryptedBytes[nonceSize:]
	plainBytes, err := aead.Open(nil, nonce, cipherText, nil) // #nosec G407
	if err != nil {
		err = kcore.Wrap(err, "error decrypting")
		return "", err
	}
	return string(plainBytes), nil
}
