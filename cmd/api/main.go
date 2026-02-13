// Package main Auto-Store API
// @title Auto-Store API
// @version 1.0
// @description Production-ready RESTful API for an auto-parts e-commerce platform.
// @host localhost:8080
// @BasePath /
// @schemes http https
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auto-store-api/internal/config"
	"auto-store-api/internal/database"
	"auto-store-api/internal/router"
	"auto-store-api/internal/validators"
	"auto-store-api/pkg/cache"
	"auto-store-api/pkg/logger"

	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	if err := logger.Init(cfg.Server.Mode); err != nil {
		panic(err)
	}
	defer logger.Sync()

	validators.Init()

	// Use retry when DB_HOST suggests Docker (postgres/redis service names)
	useRetry := cfg.Database.Host == "postgres" || cfg.Database.Host == "db"
	if useRetry {
		if err := database.ConnectWithRetry(cfg.Database.DSN, logger.Log, 10, 2*time.Second); err != nil {
			logger.Log.Fatal("database connection failed after retries", zap.Error(err))
		}
	} else {
		if err := database.Connect(cfg.Database.DSN, logger.Log); err != nil {
			logger.Log.Fatal("database connection failed", zap.Error(err))
		}
	}

	if err := cache.Connect(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB); err != nil {
		logger.Log.Warn("redis connection failed (sessions/rate limit may be affected)", zap.Error(err))
	}

	r := router.Setup(cfg, database.DB, logger.Log)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Log.Info("server starting", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Error("server shutdown error", zap.Error(err))
	}
	logger.Log.Info("server stopped")
}
