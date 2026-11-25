package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/k057ya/gophermart-loyalty/internal/model"
	"github.com/k057ya/gophermart-loyalty/internal/repository"
	"go.uber.org/zap"
)

func Login(resp http.ResponseWriter, req *http.Request, storage *repository.DBStorage, log *zap.SugaredLogger) {
	credentials := &model.Credentials{}
	err := json.NewDecoder(req.Body).Decode(credentials)
	if err != nil {
		log.Errorw("failed to decode creds from request", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := storage.UserGetByLogin(credentials.Login)
	if errors.Is(err, sql.ErrNoRows) {
		resp.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = user.Check(credentials.Password)
	if err != nil {
		resp.WriteHeader(http.StatusUnauthorized)
		return
	}


	token, err := storage.UserCreateToken(req.Context(), &user)
	if err != nil {
		log.Errorw("failed to create auth token", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.Header().Add("Authorization", "Bearer " + token)
	resp.WriteHeader(http.StatusOK)
}
