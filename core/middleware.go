package core

import (
	"errors"
	"net/http"

	"golang.org/x/exp/slog"
)

func RecoverMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rcv := recover(); rcv != nil {
				switch err := rcv.(type) {
				case error:
					if errors.Is(err, http.ErrAbortHandler) {
						panic(rcv)
					}
					slog.Error(err.Error())
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				default:
					panic(err)
				}
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
