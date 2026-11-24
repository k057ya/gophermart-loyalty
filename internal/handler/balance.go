package handler

import (
	"net/http"

	"github.com/k057ya/gophermart-loyalty/internal/repository"
	"go.uber.org/zap"
)

func Balance(resp http.ResponseWriter, req *http.Request, storage *repository.DBStorage, log *zap.SugaredLogger) {

}
