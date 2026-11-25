package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/k057ya/gophermart-loyalty/internal/repository"
	"go.uber.org/zap"
)

type authUserID struct{}

func GetAuthenticatedUserID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(authUserID{}).(int64)
	return id, ok
}

func Auth(storage *repository.DBStorage, log *zap.SugaredLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			creds := strings.Fields(r.Header.Get("Authorization"))

			if len(creds) != 2 || creds[0] != "Bearer" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			user, err := storage.UserGetByToken(r.Context(), creds[1])
			if errors.Is(err, sql.ErrNoRows) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if err != nil {
				log.Errorw("failed to get user by token", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			ctx := context.WithValue(r.Context(), authUserID{}, user.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
