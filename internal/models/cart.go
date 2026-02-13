package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartItem struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	ProductID uuid.UUID      `gorm:"type:uuid;not null;index" json:"product_id"`
	Quantity  int            `gorm:"not null;default:1" json:"quantity"`
	AddedAt   time.Time      `gorm:"column:added_at" json:"added_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

func (CartItem) TableName() string { return "shopping_cart" }

func (c *CartItem) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.AddedAt.IsZero() {
		c.AddedAt = time.Now()
	}
	return nil
}

type WishlistItem struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	ProductID uuid.UUID      `gorm:"type:uuid;not null;index" json:"product_id"`
	AddedAt   time.Time      `gorm:"column:added_at" json:"added_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

func (WishlistItem) TableName() string { return "wishlist" }

func (w *WishlistItem) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	if w.AddedAt.IsZero() {
		w.AddedAt = time.Now()
	}
	return nil
}
