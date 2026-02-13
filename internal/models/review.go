package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Review struct {
	ID               uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	ProductID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"product_id"`
	UserID           uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Rating           int            `gorm:"not null" json:"rating"`
	Title            string         `gorm:"" json:"title"`
	Comment         string         `gorm:"type:text" json:"comment"`
	VerifiedPurchase bool           `gorm:"column:verified_purchase;default:false" json:"verified_purchase"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`

	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Review) TableName() string { return "reviews" }

func (r *Review) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
