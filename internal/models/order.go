package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusProcessing OrderStatus = "processing"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

type PaymentStatus string

const (
	PaymentPending PaymentStatus = "pending"
	PaymentPaid    PaymentStatus = "paid"
	PaymentFailed  PaymentStatus = "failed"
	PaymentRefunded PaymentStatus = "refunded"
)

type Order struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	UserID             uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	OrderNumber        string         `gorm:"uniqueIndex;not null" json:"order_number"`
	Status             OrderStatus    `gorm:"type:varchar(20);default:'pending'" json:"status"`
	Subtotal           float64        `gorm:"type:decimal(12,2);not null" json:"subtotal"`
	Tax                float64        `gorm:"type:decimal(12,2);default:0" json:"tax"`
	ShippingCost       float64        `gorm:"column:shipping_cost;type:decimal(12,2);default:0" json:"shipping_cost"`
	Total              float64        `gorm:"type:decimal(12,2);not null" json:"total"`
	ShippingAddressID  uuid.UUID      `gorm:"type:uuid;column:shipping_address_id" json:"shipping_address_id"`
	BillingAddressID   uuid.UUID      `gorm:"type:uuid;column:billing_address_id" json:"billing_address_id"`
	PaymentMethod      string         `gorm:"column:payment_method" json:"payment_method"`
	PaymentStatus      PaymentStatus  `gorm:"column:payment_status;type:varchar(20);default:'pending'" json:"payment_status"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`

	User             User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	OrderItems       []OrderItem   `gorm:"foreignKey:OrderID" json:"order_items,omitempty"`
	ShippingAddress  *Address      `gorm:"foreignKey:ShippingAddressID" json:"shipping_address,omitempty"`
	BillingAddress   *Address      `gorm:"foreignKey:BillingAddressID" json:"billing_address,omitempty"`
}

func (Order) TableName() string { return "orders" }

func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	if o.OrderNumber == "" {
		o.OrderNumber = generateOrderNumber()
	}
	return nil
}

type OrderItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null;index" json:"order_id"`
	ProductID uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	Quantity  int       `gorm:"not null" json:"quantity"`
	UnitPrice float64   `gorm:"column:unit_price;type:decimal(12,2);not null" json:"unit_price"`
	TotalPrice float64  `gorm:"column:total_price;type:decimal(12,2);not null" json:"total_price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

func (OrderItem) TableName() string { return "order_items" }

func (oi *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if oi.ID == uuid.Nil {
		oi.ID = uuid.New()
	}
	return nil
}

func generateOrderNumber() string {
	return "ORD-" + uuid.New().String()[:8]
}
