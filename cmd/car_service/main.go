package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	router "github.com/InatoInato/car_service.git/internal"
	"github.com/InatoInato/car_service.git/internal/config"
	"github.com/InatoInato/car_service.git/internal/db"
	"github.com/InatoInato/car_service.git/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg := config.Load()

	logger.Info(
		"postgres configuration",
		"host", cfg.Postgres.Host,
		"port", cfg.Postgres.Port,
		"user", cfg.Postgres.User,
		"database", cfg.Postgres.Database,
		"sslmode", cfg.Postgres.SSLMode,
	)

	logger.Info(
		"connecting to postgres",
		"dsn",
		fmt.Sprintf(
			"postgres://%s:*****@%s:%s/%s?sslmode=%s",
			cfg.Postgres.User,
			cfg.Postgres.Host,
			cfg.Postgres.Port,
			cfg.Postgres.Database,
			cfg.Postgres.SSLMode,
		),
	)

	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.Database,
		cfg.Postgres.SSLMode,
	)

	dbPool, err := pgxpool.New(initCtx, dsn)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(initCtx); err != nil {
		logger.Error("database ping failed", "error", err)
		os.Exit(1)
	}

	store := db.New(dbPool)
	carService := service.NewCarService(store)
	r := router.New(logger, carService)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		logger.Info("HTTP server started", "addr", server.Addr)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}