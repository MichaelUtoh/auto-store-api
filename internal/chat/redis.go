package chat

import (
	"context"
	"encoding/json"
	"strings"

	"auto-store-api/internal/models"
	"auto-store-api/pkg/cache"

	"github.com/google/uuid"
)

// RedisPublisher publishes chat events to Redis and the local hub.
type RedisPublisher struct {
	Hub *Hub
}

func (p *RedisPublisher) PublishMessage(conversationID uuid.UUID, message *models.ChatMessage) {
	if p == nil || p.Hub == nil {
		return
	}
	payload, err := json.Marshal(WSOutbound{
		Type:    "message.new",
		Message: message,
	})
	if err != nil {
		return
	}
	p.Hub.Broadcast(conversationID, payload)
	if cache.Client == nil {
		return
	}
	channel := redisChannelPrefix + conversationID.String()
	_ = cache.Client.Publish(context.Background(), channel, payload).Err()
}

// StartRedisSubscriber listens for chat events from other API instances.
func StartRedisSubscriber(hub *Hub) {
	if cache.Client == nil || hub == nil {
		return
	}
	go func() {
		pubsub := cache.Client.PSubscribe(context.Background(), redisChannelPrefix+"*")
		ch := pubsub.Channel()
		for msg := range ch {
			if msg == nil {
				continue
			}
			idStr := strings.TrimPrefix(msg.Channel, redisChannelPrefix)
			convID, err := uuid.Parse(idStr)
			if err != nil {
				continue
			}
			hub.Broadcast(convID, []byte(msg.Payload))
		}
	}()
}

// Ensure RedisPublisher implements the interface at compile time.
var _ interface {
	PublishMessage(conversationID uuid.UUID, message *models.ChatMessage)
} = (*RedisPublisher)(nil)
