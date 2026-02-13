package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AddressType string

const (
	AddressShipping AddressType = "shipping"
	AddressBilling  AddressType = "billing"
)

type Address struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	UserID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Type       AddressType    `gorm:"type:varchar(20);not null" json:"type"`
	Street     string         `gorm:"not null" json:"street"`
	City       string         `gorm:"not null" json:"city"`
	State      string         `gorm:"" json:"state"`
	PostalCode string         `gorm:"column:postal_code" json:"postal_code"`
	Country    string         `gorm:"not null" json:"country"`
	IsDefault  bool           `gorm:"column:is_default;default:false" json:"is_default"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Address) TableName() string { return "addresses" }

func (a *Address) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
