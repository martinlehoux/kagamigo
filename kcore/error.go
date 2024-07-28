package kcore

import (
	"errors"
	"fmt"
)

func Wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

func Expect(err error, msg string) {
	if err != nil {
		if msg != "" {
			err = Wrap(err, msg)
		}
		panic(err)
	}
}

var ErrAssert = errors.New("assertion error")

func Assert(cond bool, msg string) {
	if !cond {
		err := ErrAssert
		if msg != "" {
			err = fmt.Errorf("%w: %s", err, msg)
		}
		panic(err)
	}
}
