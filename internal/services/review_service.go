package services

import (
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrAlreadyReviewed = errors.New("you have already reviewed this product")

type ReviewService struct {
	repo       *repositories.ReviewRepository
	orderRepo  *repositories.OrderRepository
	productRepo *repositories.ProductRepository
	db         *gorm.DB
}

func NewReviewService(
	repo *repositories.ReviewRepository,
	orderRepo *repositories.OrderRepository,
	productRepo *repositories.ProductRepository,
	db *gorm.DB,
) *ReviewService {
	return &ReviewService{repo: repo, orderRepo: orderRepo, productRepo: productRepo, db: db}
}

func (s *ReviewService) Create(productID, userID uuid.UUID, rating int, title, comment string) (*models.Review, error) {
	hasReviewed, _ := s.repo.UserHasReviewed(productID, userID)
	if hasReviewed {
		return nil, ErrAlreadyReviewed
	}
	verified := s.userPurchasedProduct(userID, productID)
	r := &models.Review{
		ProductID:         productID,
		UserID:            userID,
		Rating:            rating,
		Title:             title,
		Comment:           comment,
		VerifiedPurchase:  verified,
	}
	if err := s.repo.Create(r); err != nil {
		return nil, err
	}
	return r, nil
}

func (s *ReviewService) GetByProductID(productID uuid.UUID, page, limit int) ([]models.Review, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.repo.GetByProductID(productID, offset, limit)
}

func (s *ReviewService) userPurchasedProduct(userID, productID uuid.UUID) bool {
	var count int64
	s.db.Model(&models.OrderItem{}).Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.user_id = ? AND order_items.product_id = ? AND orders.status IN ?",
			userID, productID, []string{"shipped", "delivered"}).Count(&count)
	return count > 0
}
