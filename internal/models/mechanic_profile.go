package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MechanicProfileStatus string

const (
	MechanicStatusPending   MechanicProfileStatus = "pending"
	MechanicStatusVerified  MechanicProfileStatus = "verified"
	MechanicStatusSuspended MechanicProfileStatus = "suspended"
	MechanicStatusRejected  MechanicProfileStatus = "rejected"
)

// IsValidMechanicProfileStatus reports whether s is a known profile status.
func IsValidMechanicProfileStatus(s MechanicProfileStatus) bool {
	switch s {
	case MechanicStatusPending, MechanicStatusVerified, MechanicStatusSuspended, MechanicStatusRejected:
		return true
	default:
		return false
	}
}

type MechanicDocumentType string

const (
	MechanicDocLicense       MechanicDocumentType = "license"
	MechanicDocInsurance     MechanicDocumentType = "insurance"
	MechanicDocCertification MechanicDocumentType = "certification"
	MechanicDocOther         MechanicDocumentType = "other"
)

type MechanicDocumentStatus string

const (
	MechanicDocStatusPending  MechanicDocumentStatus = "pending"
	MechanicDocStatusApproved MechanicDocumentStatus = "approved"
	MechanicDocStatusRejected MechanicDocumentStatus = "rejected"
)

// MechanicProfile stores installer identity, service area, and verification state.
type MechanicProfile struct {
	ID              uuid.UUID             `gorm:"type:uuid;primary_key" json:"id"`
	UserID          uuid.UUID             `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	BusinessName    string                `gorm:"column:business_name;not null" json:"business_name"`
	Bio             string                `gorm:"type:text" json:"bio"`
	Phone           string                `gorm:"" json:"phone"`
	Street          string                `gorm:"" json:"street"`
	City            string                `gorm:"not null" json:"city"`
	State           string                `gorm:"not null" json:"state"`
	PostalCode      string                `gorm:"column:postal_code;not null" json:"postal_code"`
	Country         string                `gorm:"not null;default:'US'" json:"country"`
	Latitude        *float64              `gorm:"type:decimal(10,7)" json:"latitude,omitempty"`
	Longitude       *float64              `gorm:"type:decimal(10,7)" json:"longitude,omitempty"`
	ServiceRadiusKm float64               `gorm:"column:service_radius_km;type:decimal(8,2);default:25" json:"service_radius_km"`
	Status          MechanicProfileStatus `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	RatingAvg       float64               `gorm:"column:rating_avg;type:decimal(3,2);default:0" json:"rating_avg"`
	RatingCount     int                   `gorm:"column:rating_count;default:0" json:"rating_count"`
	VerifiedAt      *time.Time            `gorm:"column:verified_at" json:"verified_at,omitempty"`
	SuspendedAt     *time.Time            `gorm:"column:suspended_at" json:"suspended_at,omitempty"`
	RejectionReason string                `gorm:"column:rejection_reason;type:text" json:"rejection_reason,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	DeletedAt       gorm.DeletedAt        `gorm:"index" json:"-"`

	User      User               `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Documents []MechanicDocument `gorm:"foreignKey:MechanicProfileID" json:"documents,omitempty"`
}

func (MechanicProfile) TableName() string { return "mechanic_profiles" }

func (m *MechanicProfile) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if m.ServiceRadiusKm <= 0 {
		m.ServiceRadiusKm = 25
	}
	if m.Country == "" {
		m.Country = "US"
	}
	return nil
}

func (m *MechanicProfile) IsVerified() bool {
	return m.Status == MechanicStatusVerified
}

// MechanicDocument holds uploaded verification artifacts for a mechanic profile.
type MechanicDocument struct {
	ID                uuid.UUID              `gorm:"type:uuid;primary_key" json:"id"`
	MechanicProfileID uuid.UUID              `gorm:"type:uuid;not null;index" json:"mechanic_profile_id"`
	DocumentType      MechanicDocumentType   `gorm:"column:document_type;type:varchar(30);not null" json:"document_type"`
	URL               string                 `gorm:"not null" json:"url"`
	FileName          string                 `gorm:"column:file_name" json:"file_name"`
	Status            MechanicDocumentStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	ReviewNotes       string                 `gorm:"column:review_notes;type:text" json:"review_notes,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	DeletedAt         gorm.DeletedAt         `gorm:"index" json:"-"`
}

func (MechanicDocument) TableName() string { return "mechanic_documents" }

func (d *MechanicDocument) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
