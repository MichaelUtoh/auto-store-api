package services

import (
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WishlistService struct {
	repo *repositories.WishlistRepository
	db   *gorm.DB
}

func NewWishlistService(repo *repositories.WishlistRepository, db *gorm.DB) *WishlistService {
	return &WishlistService{repo: repo, db: db}
}

func (s *WishlistService) GetWishlist(userID uuid.UUID) ([]models.WishlistItem, error) {
	return s.repo.GetByUserID(userID)
}

func (s *WishlistService) Add(userID, productID uuid.UUID) (*models.WishlistItem, error) {
	exists, _ := s.repo.Exists(userID, productID)
	if exists {
		return nil, nil
	}
	item := &models.WishlistItem{UserID: userID, ProductID: productID}
	if err := s.repo.Add(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *WishlistService) Remove(userID, productID uuid.UUID) error {
	return s.repo.Remove(userID, productID)
}
