package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VehicleCompatibility is a canonical vehicle fitment record in the catalog (no product_id).
type VehicleCompatibility struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Make          string    `gorm:"not null;index" json:"make"`
	Model         string    `gorm:"not null;index" json:"model"`
	Generation    string    `gorm:"not null;default:''" json:"generation"`
	YearStart     int       `gorm:"column:year_start" json:"year_start"`
	YearEnd       int       `gorm:"column:year_end" json:"year_end"`
	Engine        string    `gorm:"not null;default:''" json:"engine"`
	Trim          string    `gorm:"not null;default:''" json:"trim"`
	MarketVariant string    `gorm:"column:market_variant;not null;default:nigeria" json:"market_variant"`
	Notes         string    `gorm:"type:text;not null;default:''" json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	Products []Product `gorm:"many2many:product_vehicle_compatibilities;" json:"-"`
}

func (VehicleCompatibility) TableName() string { return "vehicle_compatibilities" }

func (v *VehicleCompatibility) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

// ProductVehicleCompatibility links products to catalog compatibilities (M2M junction).
type ProductVehicleCompatibility struct {
	ProductID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"product_id"`
	VehicleCompatibilityID uuid.UUID `gorm:"type:uuid;primaryKey" json:"vehicle_compatibility_id"`
	Notes                  string    `gorm:"type:text;not null;default:''" json:"notes,omitempty"`
}

func (ProductVehicleCompatibility) TableName() string { return "product_vehicle_compatibilities" }
