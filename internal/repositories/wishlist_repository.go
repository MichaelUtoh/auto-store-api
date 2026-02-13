package repositories

import (
	"auto-store-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WishlistRepository struct {
	db *gorm.DB
}

func NewWishlistRepository(db *gorm.DB) *WishlistRepository {
	return &WishlistRepository{db: db}
}

func (r *WishlistRepository) GetByUserID(userID uuid.UUID) ([]models.WishlistItem, error) {
	var items []models.WishlistItem
	err := r.db.Where("user_id = ?", userID).Preload("Product").Find(&items).Error
	return items, err
}

func (r *WishlistRepository) Exists(userID, productID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.WishlistItem{}).Where("user_id = ? AND product_id = ?", userID, productID).Count(&count).Error
	return count > 0, err
}

func (r *WishlistRepository) Add(item *models.WishlistItem) error {
	return r.db.Create(item).Error
}

func (r *WishlistRepository) Remove(userID, productID uuid.UUID) error {
	return r.db.Where("user_id = ? AND product_id = ?", userID, productID).Delete(&models.WishlistItem{}).Error
}
