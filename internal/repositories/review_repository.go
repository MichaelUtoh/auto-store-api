package repositories

import (
	"auto-store-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) Create(review *models.Review) error {
	return r.db.Create(review).Error
}

func (r *ReviewRepository) GetByProductID(productID uuid.UUID, offset, limit int) ([]models.Review, int64, error) {
	var reviews []models.Review
	var total int64
	q := r.db.Model(&models.Review{}).Where("product_id = ?", productID)
	q.Count(&total)
	err := q.Preload("User").Offset(offset).Limit(limit).Order("created_at DESC").Find(&reviews).Error
	return reviews, total, err
}

func (r *ReviewRepository) UserHasReviewed(productID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.Review{}).Where("product_id = ? AND user_id = ?", productID, userID).Count(&count).Error
	return count > 0, err
}
