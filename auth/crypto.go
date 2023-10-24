package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/martinlehoux/kagamigo/core"
)

var (
	ErrEncryptedTooShort = errors.New("encrypted text is too short")
)

func encrypt(secret []byte, plainText string) string {
	block, err := aes.NewCipher(secret)
	core.Expect(err, "error creating AES cipher")

	aead, err := cipher.NewGCM(block)
	core.Expect(err, "error creating AEAD")

	nonce := make([]byte, aead.NonceSize())
	_, err = rand.Read(nonce)
	core.Expect(err, "error generating nonce")

	return base64.URLEncoding.EncodeToString(aead.Seal(nonce, nonce, []byte(plainText), nil))
}

func decrypt(secret []byte, encryptedText string) (string, error) {
	encryptedBytes, err := base64.URLEncoding.DecodeString(encryptedText)
	if err != nil {
		err = core.Wrap(err, "error decoding base64")
		return "", err
	}
	block, err := aes.NewCipher(secret)
	core.Expect(err, "error creating AES cipher")

	aead, err := cipher.NewGCM(block)
	core.Expect(err, "error creating AEAD")

	nonceSize := aead.NonceSize()
	if len(encryptedBytes) < nonceSize {
		return "", ErrEncryptedTooShort
	}
	nonce, cipherText := encryptedBytes[:nonceSize], encryptedBytes[nonceSize:]
	plainBytes, err := aead.Open(nil, nonce, cipherText, nil)
	if err != nil {
		err = core.Wrap(err, "error decrypting")
		return "", err
	}
	return string(plainBytes), nil
}
