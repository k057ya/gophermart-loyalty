package handler

import (
	"net/http"

	"github.com/k057ya/gophermart-loyalty/internal/repository"
	"go.uber.org/zap"
)

func Withdraw(resp http.ResponseWriter, req *http.Request, storage *repository.DBStorage, log *zap.SugaredLogger) {

}
