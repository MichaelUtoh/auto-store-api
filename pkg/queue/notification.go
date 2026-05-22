package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

const NotificationSendQueue = "notifications:send"

// SendJob is pushed to Redis for the worker to deliver external notifications (email, etc.).
type SendJob struct {
	NotificationID uuid.UUID `json:"notification_id"`
}

func EnqueueNotification(ctx context.Context, client *redis.Client, notificationID uuid.UUID) error {
	if client == nil {
		return redis.Nil
	}
	data, err := json.Marshal(SendJob{NotificationID: notificationID})
	if err != nil {
		return err
	}
	return client.LPush(ctx, NotificationSendQueue, data).Err()
}

func DequeueNotification(ctx context.Context, client *redis.Client, timeout time.Duration) (*SendJob, error) {
	if client == nil {
		return nil, redis.Nil
	}
	result, err := client.BRPop(ctx, timeout, NotificationSendQueue).Result()
	if err != nil {
		return nil, err
	}
	if len(result) < 2 {
		return nil, redis.Nil
	}
	var job SendJob
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, err
	}
	return &job, nil
}
