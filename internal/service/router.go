package service

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/k057ya/gophermart-loyalty/internal/handler"
	"github.com/k057ya/gophermart-loyalty/internal/middleware"
	"github.com/k057ya/gophermart-loyalty/internal/repository"
	"go.uber.org/zap"
)

func NewRouter(storage *repository.DBStorage, log *zap.SugaredLogger) *chi.Mux {
	router := chi.NewRouter()
	// Add Middlewares:
	// - chi middleware to strip trailing slash
	router.Use(chimiddleware.StripSlashes)

	// - gzip compression middleware
	router.Use(middleware.Compress(log.Named("gzip")))

	// - logger middleware for all routes
	router.Use(middleware.Log(log.Named("http")))

	handlerLog := log.Named("handler")
	// wrap all handlers to inject storage and logger
	wrap := func(controller func(http.ResponseWriter, *http.Request, *repository.DBStorage, *zap.SugaredLogger)) http.HandlerFunc {
		return func(w http.ResponseWriter, req *http.Request) {
			controller(w, req, storage, handlerLog)
		}
	}

	router.Route("/api/user", func(r chi.Router) {
		// Register
		r.Post("/register", wrap(handler.Register))
		// Login
		r.Post("/login", wrap(handler.Login))

		r.Group(func(r chi.Router) {
			// Under auth
			r.Use(middleware.Auth(storage, log.Named("auth")))
			// Orders create
			r.Post("/orders", wrap(handler.Order))
			// Orders list
			r.Get("/orders", wrap(handler.OrdersList))

			r.Route("/balance", func(r chi.Router) {
				// balance get
				r.Get("/", wrap(handler.Balance))
				// withdraw
				r.Post("/withdraw", wrap(handler.Withdraw))
			})

			// Withdrawals list
			r.Get("/withdrawals", wrap(handler.WithdrawalsList))
		})
	})
	return router
}
