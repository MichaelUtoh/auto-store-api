package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auto-store-api/internal/config"
	"auto-store-api/internal/database"
	"auto-store-api/internal/repositories"
	"auto-store-api/internal/services"
	"auto-store-api/pkg/cache"
	"auto-store-api/pkg/email"
	"auto-store-api/pkg/logger"
	"auto-store-api/pkg/queue"

	"github.com/go-redis/redis/v8"
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

	useRetry := cfg.Database.Host == "postgres" || cfg.Database.Host == "db"
	if useRetry {
		err = database.ConnectWithRetry(cfg.Database.DSN, logger.Log, 10, 2*time.Second)
	} else {
		err = database.Connect(cfg.Database.DSN, logger.Log)
	}
	if err != nil {
		logger.Log.Fatal("database connection failed", zap.Error(err))
	}

	var redisClient *redis.Client
	if err := cache.Connect(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB); err != nil {
		logger.Log.Warn("redis connection failed; worker will poll database only", zap.Error(err))
	} else {
		redisClient = cache.Client
	}

	notifRepo := repositories.NewNotificationRepository(database.DB)
	userRepo := repositories.NewUserRepository(database.DB)
	emailSender := email.NewSender(cfg.Email)
	notifSvc := services.NewNotificationService(notifRepo, userRepo, redisClient, emailSender, cfg, logger.Log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dequeueTimeout := time.Duration(cfg.Notifications.DequeueTimeoutSec) * time.Second
	if dequeueTimeout <= 0 {
		dequeueTimeout = 5 * time.Second
	}

	go pollPending(ctx, notifSvc)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		cancel()
	}()

	logger.Log.Info("notification worker started")

	for {
		if ctx.Err() != nil {
			logger.Log.Info("notification worker stopped")
			return
		}

		if redisClient == nil {
			time.Sleep(dequeueTimeout)
			continue
		}

		job, err := queue.DequeueNotification(ctx, redisClient, dequeueTimeout)
		if err != nil {
			if ctx.Err() != nil {
				continue
			}
			if err != redis.Nil {
				logger.Log.Warn("dequeue error", zap.Error(err))
			}
			continue
		}

		if err := notifSvc.ProcessDelivery(ctx, job.NotificationID); err != nil {
			logger.Log.Warn("delivery failed",
				zap.String("notification_id", job.NotificationID.String()),
				zap.Error(err),
			)
			_ = notifSvc.RequeuePending(ctx, job.NotificationID)
		}
	}
}

func pollPending(ctx context.Context, notifSvc *services.NotificationService) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := notifSvc.PollAndProcessPending(ctx, 50)
			if err != nil {
				logger.Log.Warn("poll pending notifications failed", zap.Error(err))
			} else if n > 0 {
				logger.Log.Info("processed pending notifications from database poll", zap.Int("count", n))
			}
		}
	}
}
