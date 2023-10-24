package kcore

import (
	"encoding/base64"

	"github.com/gofrs/uuid"
)

type ID struct {
	uuid.UUID
}

func NewID() ID {
	id, err := uuid.NewV4()
	Expect(err, "error generating uuid")

	return ID{id}
}

func (id ID) String() string {
	return base64.RawURLEncoding.EncodeToString(id.Bytes())
}

func ParseID(value string) (ID, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return ID{}, Wrap(err, "error decoding value")
	}
	id, err := uuid.FromBytes(bytes)
	if err != nil {
		return ID{}, Wrap(err, "error parsing uuid")
	}
	return ID{id}, nil
}
