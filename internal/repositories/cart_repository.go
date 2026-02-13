package repositories

import (
	"auto-store-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) *CartRepository {
	return &CartRepository{db: db}
}

func (r *CartRepository) GetByUserID(userID uuid.UUID) ([]models.CartItem, error) {
	var items []models.CartItem
	err := r.db.Where("user_id = ?", userID).Preload("Product").Find(&items).Error
	return items, err
}

func (r *CartRepository) GetItem(userID, productID uuid.UUID) (*models.CartItem, error) {
	var item models.CartItem
	err := r.db.Where("user_id = ? AND product_id = ?", userID, productID).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CartRepository) Add(item *models.CartItem) error {
	return r.db.Create(item).Error
}

func (r *CartRepository) UpdateQuantity(id, userID uuid.UUID, quantity int) error {
	return r.db.Model(&models.CartItem{}).Where("id = ? AND user_id = ?", id, userID).Update("quantity", quantity).Error
}

func (r *CartRepository) Delete(id, userID uuid.UUID) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.CartItem{}).Error
}

func (r *CartRepository) Clear(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.CartItem{}).Error
}

func (r *CartRepository) GetItemByID(id, userID uuid.UUID) (*models.CartItem, error) {
	var item models.CartItem
	err := r.db.Where("id = ? AND user_id = ?", id, userID).Preload("Product").First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}
