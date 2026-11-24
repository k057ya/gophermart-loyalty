package middleware

import (
	"net/http"

	"go.uber.org/zap"
)

func Auth(log *zap.SugaredLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO check auth
		})
	}
}
