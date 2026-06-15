package repositories

import (
	"auto-store-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

func (r *InventoryRepository) CreateMovement(m *models.StockMovement) error {
	return r.db.Create(m).Error
}

func (r *InventoryRepository) ListMovements(productID uuid.UUID, offset, limit int) ([]models.StockMovement, int64, error) {
	var movements []models.StockMovement
	var total int64
	q := r.db.Model(&models.StockMovement{}).Where("product_id = ?", productID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&movements).Error
	return movements, total, err
}

func (r *InventoryRepository) ListLowStock(offset, limit int, vendorID *uuid.UUID) ([]models.Product, int64, error) {
	var products []models.Product
	var total int64
	q := r.db.Model(&models.Product{}).
		Where("stock_quantity <= low_stock_threshold")
	if vendorID != nil {
		q = q.Where("vendor_id = ?", *vendorID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Preload("Images").Order("stock_quantity ASC").Offset(offset).Limit(limit).Find(&products).Error
	return products, total, err
}

func (r *InventoryRepository) ListUnnotifiedLowStock(limit int) ([]models.Product, error) {
	var products []models.Product
	err := r.db.Where("stock_quantity <= low_stock_threshold AND low_stock_notified = ?", false).
		Limit(limit).
		Find(&products).Error
	return products, err
}

func (r *InventoryRepository) BulkSetThreshold(threshold int, categoryID *uuid.UUID, productIDs []uuid.UUID) (int64, error) {
	q := r.db.Model(&models.Product{})
	if len(productIDs) > 0 {
		q = q.Where("id IN ?", productIDs)
	} else if categoryID != nil {
		q = q.Where(`id IN (
			SELECT product_id FROM product_categories WHERE category_id = ?
		)`, *categoryID)
	}
	res := q.Update("low_stock_threshold", threshold)
	return res.RowsAffected, res.Error
}
