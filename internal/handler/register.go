package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/k057ya/gophermart-loyalty/internal/model"
	"github.com/k057ya/gophermart-loyalty/internal/repository"
	"go.uber.org/zap"
)

func Register(resp http.ResponseWriter, req *http.Request, storage *repository.DBStorage, log *zap.SugaredLogger) {
	ctx := req.Context()
	user := &model.User{}
	err := json.NewDecoder(req.Body).Decode(&user.Credentials)
	if err != nil {
		log.Errorw("failed to decode creds from request", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	user.Password, err = user.HashPassword(user.Password)
	if err != nil {
		log.Errorw("failed to hash password", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = storage.UserCreate(ctx, user)
	if err != nil {
		if errors.Is(err, repository.ErrLoginTaken) {
			log.Errorw("failed to register user as login is taken", err)
			resp.WriteHeader(http.StatusConflict)
			return
		}
		log.Errorw("failed to store user", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	token, err := storage.UserCreateToken(ctx, user)
	if err != nil {
		log.Errorw("failed to create auth token", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.Header().Add("Authorization", "Bearer " + token)
	resp.WriteHeader(http.StatusOK)
}
