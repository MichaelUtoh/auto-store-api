package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ConversationStatus string

const (
	ConversationStatusOpen   ConversationStatus = "open"
	ConversationStatusClosed ConversationStatus = "closed"
)

type ChatContextType string

const (
	ChatContextGeneral ChatContextType = "general"
	ChatContextOrder   ChatContextType = "order"
	ChatContextProduct ChatContextType = "product"
)

type MessageSenderType string

const (
	MessageSenderCustomer MessageSenderType = "customer"
	MessageSenderAdmin    MessageSenderType = "admin"
	MessageSenderSystem   MessageSenderType = "system"
)

func IsValidConversationStatus(s ConversationStatus) bool {
	return s == ConversationStatusOpen || s == ConversationStatusClosed
}

func IsValidChatContextType(t ChatContextType) bool {
	switch t {
	case ChatContextGeneral, ChatContextOrder, ChatContextProduct:
		return true
	default:
		return false
	}
}

type Conversation struct {
	ID                  uuid.UUID          `gorm:"type:uuid;primary_key" json:"id"`
	UserID              *uuid.UUID         `gorm:"type:uuid;index:idx_conv_user_status,priority:1" json:"user_id"`
	GuestID             *uuid.UUID         `gorm:"type:uuid;index:idx_conv_guest_status,priority:1" json:"guest_id"`
	GuestEmail          string             `gorm:"column:guest_email" json:"guest_email"`
	GuestName           string             `gorm:"column:guest_name" json:"guest_name"`
	Status              ConversationStatus `gorm:"type:varchar(20);not null;default:'open';index:idx_conv_user_status,priority:2;index:idx_conv_guest_status,priority:2" json:"status"`
	ContextType         *ChatContextType   `gorm:"type:varchar(20)" json:"context_type"`
	ContextID           *uuid.UUID         `gorm:"type:uuid" json:"context_id"`
	LastMessageAt       time.Time          `gorm:"index" json:"last_message_at"`
	CustomerLastReadAt  *time.Time         `gorm:"column:customer_last_read_at" json:"-"`
	AdminLastReadAt     *time.Time         `gorm:"column:admin_last_read_at" json:"-"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
	DeletedAt           gorm.DeletedAt     `gorm:"index" json:"-"`

	Messages []ChatMessage `gorm:"foreignKey:ConversationID" json:"-"`
}

func (Conversation) TableName() string { return "conversations" }

func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.LastMessageAt.IsZero() {
		c.LastMessageAt = time.Now()
	}
	return nil
}

type ChatMessage struct {
	ID             uuid.UUID         `gorm:"type:uuid;primary_key" json:"id"`
	ConversationID uuid.UUID         `gorm:"type:uuid;not null;index" json:"conversation_id"`
	SenderType     MessageSenderType `gorm:"type:varchar(20);not null" json:"sender_type"`
	SenderUserID   *uuid.UUID        `gorm:"type:uuid" json:"sender_user_id"`
	Body           string            `gorm:"type:text;not null" json:"body"`
	CreatedAt      time.Time         `json:"created_at"`
}

func (ChatMessage) TableName() string { return "chat_messages" }

func (m *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
