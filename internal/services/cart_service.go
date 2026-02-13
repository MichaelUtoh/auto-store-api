package services

import (
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrProductNotFound = errors.New("product not found")
var ErrInsufficientStock = errors.New("insufficient stock")

type CartService struct {
	cartRepo    *repositories.CartRepository
	productRepo *repositories.ProductRepository
	db          *gorm.DB
}

func NewCartService(cartRepo *repositories.CartRepository, productRepo *repositories.ProductRepository, db *gorm.DB) *CartService {
	return &CartService{cartRepo: cartRepo, productRepo: productRepo, db: db}
}

func (s *CartService) GetCart(userID uuid.UUID) ([]models.CartItem, error) {
	return s.cartRepo.GetByUserID(userID)
}

func (s *CartService) AddItem(userID, productID uuid.UUID, quantity int) (*models.CartItem, error) {
	product, err := s.productRepo.GetByID(productID)
	if err != nil || product == nil {
		return nil, ErrProductNotFound
	}
	if product.StockQuantity < quantity {
		return nil, ErrInsufficientStock
	}
	existing, err := s.cartRepo.GetItem(userID, productID)
	if err == nil {
		existing.Quantity += quantity
		if existing.Quantity > product.StockQuantity {
			existing.Quantity = product.StockQuantity
		}
		if err := s.db.Model(&models.CartItem{}).Where("id = ?", existing.ID).Update("quantity", existing.Quantity).Error; err != nil {
			return nil, err
		}
		return existing, nil
	}
	item := &models.CartItem{
		UserID:    userID,
		ProductID: productID,
		Quantity:  quantity,
	}
	if err := s.cartRepo.Add(item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *CartService) UpdateItem(userID, itemID uuid.UUID, quantity int) error {
	item, err := s.cartRepo.GetItemByID(itemID, userID)
	if err != nil {
		return err
	}
	product, _ := s.productRepo.GetByID(item.ProductID)
	if product != nil && quantity > product.StockQuantity {
		quantity = product.StockQuantity
	}
	return s.cartRepo.UpdateQuantity(itemID, userID, quantity)
}

func (s *CartService) RemoveItem(userID, itemID uuid.UUID) error {
	return s.cartRepo.Delete(itemID, userID)
}

func (s *CartService) Clear(userID uuid.UUID) error {
	return s.cartRepo.Clear(userID)
}
