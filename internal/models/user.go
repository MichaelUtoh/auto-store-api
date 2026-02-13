package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string

const (
	RoleAdmin    Role = "ADMIN"
	RoleVendor   Role = "VENDOR"
	RoleCustomer Role = "CUSTOMER"
)

type User struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Email         string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash  string         `gorm:"column:password_hash;not null" json:"-"`
	FirstName     string         `gorm:"column:first_name" json:"first_name"`
	LastName      string         `gorm:"column:last_name" json:"last_name"`
	Role          Role           `gorm:"type:varchar(20);default:'CUSTOMER'" json:"role"`
	Phone         string         `gorm:"" json:"phone"`
	EmailVerified bool           `gorm:"column:email_verified;default:false" json:"email_verified"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	Addresses []Address `gorm:"foreignKey:UserID" json:"addresses,omitempty"`
}

func (User) TableName() string { return "users" }

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
