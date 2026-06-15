package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const DefaultLowStockThreshold = 5

type StockMovementReason string

const (
	StockMovementOrder      StockMovementReason = "order"
	StockMovementRestock    StockMovementReason = "restock"
	StockMovementAdjustment StockMovementReason = "adjustment"
	StockMovementRefund     StockMovementReason = "refund"
	StockMovementCancel     StockMovementReason = "cancel"
)

type StockMovement struct {
	ID            uuid.UUID           `gorm:"type:uuid;primary_key" json:"id"`
	ProductID     uuid.UUID           `gorm:"type:uuid;not null;index" json:"product_id"`
	Delta         int                 `gorm:"not null" json:"delta"`
	QuantityAfter int                 `gorm:"column:quantity_after;not null" json:"quantity_after"`
	Reason        StockMovementReason `gorm:"type:varchar(20);not null" json:"reason"`
	ReferenceID   *uuid.UUID          `gorm:"type:uuid;column:reference_id" json:"reference_id,omitempty"`
	PerformedBy   *uuid.UUID          `gorm:"type:uuid;column:performed_by" json:"performed_by,omitempty"`
	Notes         string              `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt     time.Time           `json:"created_at"`
}

func (StockMovement) TableName() string { return "stock_movements" }

func (m *StockMovement) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
