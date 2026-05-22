package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationChannel string

const (
	NotificationChannelInApp NotificationChannel = "in_app"
	NotificationChannelEmail NotificationChannel = "email"
	NotificationChannelSMS   NotificationChannel = "sms"
	NotificationChannelPush  NotificationChannel = "push"
)

type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusQueued  NotificationStatus = "queued"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
)

// NotificationType identifies the business event that triggered the notification.
type NotificationType string

const (
	NotificationQuoteReady         NotificationType = "quote.ready"
	NotificationBookingConfirmed   NotificationType = "booking.confirmed"
	NotificationMechanicEnRoute      NotificationType = "mechanic.en_route"
	NotificationQAAnswerPosted       NotificationType = "qa.answer_posted"
	NotificationMechanicVerified     NotificationType = "mechanic.verified"
	NotificationMechanicApplyReceived NotificationType = "mechanic.apply_received"
)

// Notification is a single delivery record (one row per user + channel + idempotency key).
type Notification struct {
	ID               uuid.UUID           `gorm:"type:uuid;primary_key" json:"id"`
	UserID           uuid.UUID           `gorm:"type:uuid;not null;index" json:"user_id"`
	Type             NotificationType    `gorm:"type:varchar(50);not null;index" json:"type"`
	Channel          NotificationChannel `gorm:"type:varchar(20);not null" json:"channel"`
	Title            string              `gorm:"not null" json:"title"`
	Body             string              `gorm:"type:text;not null" json:"body"`
	Payload          string              `gorm:"type:jsonb" json:"payload,omitempty"`
	Status           NotificationStatus  `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	IdempotencyKey   string              `gorm:"column:idempotency_key;uniqueIndex;not null" json:"-"`
	ReadAt           *time.Time          `gorm:"column:read_at" json:"read_at,omitempty"`
	SentAt           *time.Time          `gorm:"column:sent_at" json:"sent_at,omitempty"`
	FailedReason     string              `gorm:"column:failed_reason;type:text" json:"-"`
	RetryCount       int                 `gorm:"column:retry_count;default:0" json:"-"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
	DeletedAt        gorm.DeletedAt      `gorm:"index" json:"-"`
}

func (Notification) TableName() string { return "notifications" }

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}

// NotificationPreference stores per-user delivery preferences.
type NotificationPreference struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	EmailEnabled bool      `gorm:"column:email_enabled;default:true" json:"email_enabled"`
	SmsEnabled   bool      `gorm:"column:sms_enabled;default:false" json:"sms_enabled"`
	PushEnabled  bool      `gorm:"column:push_enabled;default:false" json:"push_enabled"`
	InAppEnabled bool      `gorm:"column:in_app_enabled;default:true" json:"in_app_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (NotificationPreference) TableName() string { return "notification_preferences" }

func (p *NotificationPreference) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
