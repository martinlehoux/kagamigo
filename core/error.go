package core

import "fmt"

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
