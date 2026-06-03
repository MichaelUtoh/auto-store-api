package dto

import (
	"time"

	"auto-store-api/internal/models"

	"github.com/google/uuid"
)

type GuestSessionResponse struct {
	GuestID    uuid.UUID `json:"guest_id"`
	GuestToken string    `json:"guest_token"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type CreateConversationRequest struct {
	ContextType *models.ChatContextType `json:"context_type"`
	ContextID   *uuid.UUID              `json:"context_id"`
	GuestName   string                  `json:"guest_name"`
}

type UpdateConversationRequest struct {
	Status     *models.ConversationStatus `json:"status"`
	GuestEmail *string                    `json:"guest_email" binding:"omitempty,email"`
	GuestName  *string                    `json:"guest_name"`
}

type SendChatMessageRequest struct {
	Body string `json:"body" binding:"required"`
}

type LinkGuestRequest struct {
	GuestToken string `json:"guest_token" binding:"required"`
}

type LinkGuestResponse struct {
	LinkedCount      int         `json:"linked_count"`
	ConversationIDs  []uuid.UUID `json:"conversation_ids"`
}

type ConversationResponse struct {
	ID            uuid.UUID               `json:"id"`
	UserID        *uuid.UUID              `json:"user_id"`
	GuestID       *uuid.UUID              `json:"guest_id"`
	GuestEmail    string                  `json:"guest_email"`
	GuestName     string                  `json:"guest_name"`
	Status        models.ConversationStatus `json:"status"`
	ContextType   *models.ChatContextType `json:"context_type"`
	ContextID     *uuid.UUID              `json:"context_id"`
	LastMessageAt time.Time               `json:"last_message_at"`
	UnreadCount   int64                   `json:"unread_count,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
}

type ChatMessageResponse struct {
	ID             uuid.UUID              `json:"id"`
	ConversationID uuid.UUID              `json:"conversation_id"`
	SenderType     models.MessageSenderType `json:"sender_type"`
	SenderUserID   *uuid.UUID             `json:"sender_user_id"`
	Body           string                 `json:"body"`
	CreatedAt      time.Time              `json:"created_at"`
}

func ConversationToResponse(c *models.Conversation, unread int64) ConversationResponse {
	return ConversationResponse{
		ID:            c.ID,
		UserID:        c.UserID,
		GuestID:       c.GuestID,
		GuestEmail:    c.GuestEmail,
		GuestName:     c.GuestName,
		Status:        c.Status,
		ContextType:   c.ContextType,
		ContextID:     c.ContextID,
		LastMessageAt: c.LastMessageAt,
		UnreadCount:   unread,
		CreatedAt:     c.CreatedAt,
	}
}

func ChatMessageToResponse(m *models.ChatMessage) ChatMessageResponse {
	return ChatMessageResponse{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderType:     m.SenderType,
		SenderUserID:   m.SenderUserID,
		Body:           m.Body,
		CreatedAt:      m.CreatedAt,
	}
}
