package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/k057ya/gophermart-loyalty/internal/accrual"
	"github.com/k057ya/gophermart-loyalty/internal/config"
	"github.com/k057ya/gophermart-loyalty/internal/repository"
	"github.com/k057ya/gophermart-loyalty/internal/service"
)

func main() {
	// Init logger
	logger, err := service.NewLogger("info")
	if err != nil {
		fmt.Printf("error creating logger %v", err)
		return
	}
	defer logger.Sync()

	// Prepare config
	configLogger := logger.Named("config")
	if err := config.ParseFlags(os.Args[1:]); err != nil {
		// if !errors.Is(err, flag.ErrHelp) {
		// 	os.Exit(2)
		// }
		return
	}
	if err := config.ParseEnv(); err != nil {
		configLogger.Warnw("using HTTP address from flags or defaults", err)
	}
	if err := config.ServiceCfg.Validate(); err != nil {
		configLogger.Errorw("invalid configuration", err)
		_ = logger.Sync()
		os.Exit(2)
	}

	// Context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Init storage
	storage, err := repository.NewDBStorage(config.ServiceCfg.Database.Dsn, logger.Named("db"))
	if err != nil {
		logger.Errorw("error starting db storage", err)
		return
	}
	defer storage.Close()

	// Start http server
	serverLogger := logger.Named("http")
	server := &http.Server{
		Addr:    config.ServiceCfg.HTTP.String(),
		Handler: service.NewRouter(storage, serverLogger),
	}
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.ListenAndServe()
	}()

	// Run accrual client
	clientLogger := logger.Named("accrual")
	client := accrual.NewClient(
		accrual.NewHTTPClient(config.ServiceCfg.Client.BaseURL),
		storage,
		clientLogger,
	)
	clientDone := make(chan error, 1)
	go func() {
		clientDone <- accrual.Run(ctx, client)
	}()

	// Graceful shutdown
	select {
	case <-ctx.Done():
	case err := <-serverDone:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverLogger.Errorw("HTTP server failed", err)
		}
	}
	stop()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		serverLogger.Errorw("HTTP shutdown failed", err)
		if err := server.Close(); err != nil {
			serverLogger.Errorw("HTTP close failed", err)
		}
	}

	if err := <-clientDone; err != nil {
		clientLogger.Errorw("accrual client run error", err)
	}

}
