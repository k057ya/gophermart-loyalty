package handler

import (
	"net/http"

	"github.com/k057ya/gophermart-loyalty/internal/repository"
	"go.uber.org/zap"
)

func Register(resp http.ResponseWriter, req *http.Request, storage *repository.DBStorage, log *zap.SugaredLogger) {

}
