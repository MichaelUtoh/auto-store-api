package repositories

import (
	"auto-store-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(o *models.Order) error {
	return r.db.Create(o).Error
}

func (r *OrderRepository) GetByID(id uuid.UUID) (*models.Order, error) {
	var o models.Order
	err := r.db.Preload("OrderItems").Preload("OrderItems.Product").
		Preload("ShippingAddress").Preload("BillingAddress").
		First(&o, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepository) GetByPaymentReference(ref string) (*models.Order, error) {
	var o models.Order
	err := r.db.First(&o, "payment_reference = ?", ref).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepository) GetByOrderNumber(num string) (*models.Order, error) {
	var o models.Order
	err := r.db.First(&o, "order_number = ?", num).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepository) GetByUserID(userID uuid.UUID, offset, limit int) ([]models.Order, int64, error) {
	var orders []models.Order
	var total int64
	q := r.db.Model(&models.Order{}).Where("user_id = ?", userID)
	q.Count(&total)
	err := q.Preload("OrderItems").Preload("OrderItems.Product").
		Offset(offset).Limit(limit).Order("created_at DESC").Find(&orders).Error
	return orders, total, err
}

func (r *OrderRepository) ListAll(offset, limit int, status string) ([]models.Order, int64, error) {
	var orders []models.Order
	var total int64
	q := r.db.Model(&models.Order{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q.Count(&total)
	err := q.Preload("User").Preload("OrderItems").Preload("OrderItems.Product").
		Offset(offset).Limit(limit).Order("created_at DESC").Find(&orders).Error
	return orders, total, err
}

func (r *OrderRepository) Update(o *models.Order) error {
	return r.db.Save(o).Error
}

func (r *OrderRepository) AddItem(item *models.OrderItem) error {
	return r.db.Create(item).Error
}
