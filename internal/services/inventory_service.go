package services

import (
	"context"
	"errors"
	"fmt"

	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInventoryProductNotFound = errors.New("product not found")
	ErrInventoryAccessDenied    = errors.New("access denied")
	ErrInvalidStockDelta        = errors.New("stock quantity cannot be negative")
	ErrOutOfStock               = errors.New("out of stock")
)

type InventoryService struct {
	inventoryRepo *repositories.InventoryRepository
	productRepo   *repositories.ProductRepository
	userRepo      *repositories.UserRepository
	notifier      *Notifier
	db            *gorm.DB
}

func NewInventoryService(
	inventoryRepo *repositories.InventoryRepository,
	productRepo *repositories.ProductRepository,
	userRepo *repositories.UserRepository,
	notifier *Notifier,
	db *gorm.DB,
) *InventoryService {
	return &InventoryService{
		inventoryRepo: inventoryRepo,
		productRepo:   productRepo,
		userRepo:      userRepo,
		notifier:      notifier,
		db:            db,
	}
}

type AdjustStockInput struct {
	ProductID   uuid.UUID
	Delta       int
	Reason      models.StockMovementReason
	ReferenceID *uuid.UUID
	PerformedBy *uuid.UUID
	Notes       string
}

func (s *InventoryService) AdjustStock(ctx context.Context, tx *gorm.DB, in AdjustStockInput) (*models.Product, error) {
	if tx == nil {
		var product *models.Product
		err := s.db.Transaction(func(innerTx *gorm.DB) error {
			var err error
			product, err = s.adjustStockTx(ctx, innerTx, in)
			return err
		})
		return product, err
	}
	return s.adjustStockTx(ctx, tx, in)
}

func (s *InventoryService) adjustStockTx(ctx context.Context, tx *gorm.DB, in AdjustStockInput) (*models.Product, error) {
	var product models.Product
	if err := tx.First(&product, "id = ?", in.ProductID).Error; err != nil {
		return nil, ErrInventoryProductNotFound
	}

	newQty := product.StockQuantity + in.Delta
	if newQty < 0 {
		return nil, ErrInvalidStockDelta
	}

	product.StockQuantity = newQty
	if err := tx.Save(&product).Error; err != nil {
		return nil, err
	}

	movement := &models.StockMovement{
		ProductID:     product.ID,
		Delta:         in.Delta,
		QuantityAfter: newQty,
		Reason:        in.Reason,
		ReferenceID:   in.ReferenceID,
		PerformedBy:   in.PerformedBy,
		Notes:         in.Notes,
	}
	if err := tx.Create(movement).Error; err != nil {
		return nil, err
	}

	if err := s.handleThreshold(ctx, tx, &product); err != nil {
		return nil, err
	}

	return &product, nil
}

func (s *InventoryService) handleThreshold(ctx context.Context, tx *gorm.DB, product *models.Product) error {
	threshold := product.LowStockThreshold
	if threshold <= 0 {
		threshold = models.DefaultLowStockThreshold
	}

	if product.StockQuantity <= threshold {
		if !product.LowStockNotified {
			product.LowStockAlertCycle++
			product.LowStockNotified = true
			if err := tx.Model(product).Updates(map[string]interface{}{
				"low_stock_notified":    true,
				"low_stock_alert_cycle": product.LowStockAlertCycle,
			}).Error; err != nil {
				return err
			}
			go s.sendLowStockAlerts(context.Background(), product)
		}
	} else if product.LowStockNotified {
		product.LowStockNotified = false
		if err := tx.Model(product).Update("low_stock_notified", false).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *InventoryService) sendLowStockAlerts(ctx context.Context, product *models.Product) {
	if s.notifier == nil {
		return
	}
	cycle := product.LowStockAlertCycle
	admins, _ := s.userRepo.ListByRole(models.RoleAdmin)
	for _, admin := range admins {
		_ = s.notifier.LowStockAlert(ctx, admin.ID, product, cycle)
	}
	if product.VendorID != nil {
		_ = s.notifier.LowStockAlert(ctx, *product.VendorID, product, cycle)
	}
}

func (s *InventoryService) ScanLowStock(ctx context.Context) (int, error) {
	products, err := s.inventoryRepo.ListUnnotifiedLowStock(100)
	if err != nil {
		return 0, err
	}
	count := 0
	for i := range products {
		p := &products[i]
		if err := s.db.Transaction(func(tx *gorm.DB) error {
			return s.handleThreshold(ctx, tx, p)
		}); err == nil && p.LowStockNotified {
			count++
		}
	}
	return count, nil
}

func (s *InventoryService) CanAccessProduct(user *models.User, product *models.Product) bool {
	if user.Role == models.RoleAdmin {
		return true
	}
	if user.Role == models.RoleVendor && product.VendorID != nil && *product.VendorID == user.ID {
		return true
	}
	return false
}

func (s *InventoryService) GetProductForUser(user *models.User, productID uuid.UUID) (*models.Product, error) {
	product, err := s.productRepo.GetByID(productID)
	if err != nil || product == nil {
		return nil, ErrInventoryProductNotFound
	}
	if !s.CanAccessProduct(user, product) {
		return nil, ErrInventoryAccessDenied
	}
	return product, nil
}

func (s *InventoryService) ListLowStock(user *models.User, page, limit int) ([]models.Product, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit
	var vendorID *uuid.UUID
	if user.Role == models.RoleVendor {
		vendorID = &user.ID
	}
	return s.inventoryRepo.ListLowStock(offset, limit, vendorID)
}

func (s *InventoryService) ListMovements(user *models.User, productID uuid.UUID, page, limit int) ([]models.StockMovement, int64, error) {
	if _, err := s.GetProductForUser(user, productID); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.inventoryRepo.ListMovements(productID, offset, limit)
}

func (s *InventoryService) UpdateSettings(user *models.User, productID uuid.UUID, threshold int) (*models.Product, error) {
	product, err := s.GetProductForUser(user, productID)
	if err != nil {
		return nil, err
	}
	if threshold < 0 {
		return nil, fmt.Errorf("threshold must be >= 0")
	}
	product.LowStockThreshold = threshold
	if err := s.productRepo.Update(product); err != nil {
		return nil, err
	}
	return product, nil
}

type BulkThresholdInput struct {
	Threshold  int
	CategoryID *uuid.UUID
	ProductIDs []uuid.UUID
}

func (s *InventoryService) BulkSetThreshold(in BulkThresholdInput) (int64, error) {
	if in.Threshold < 0 {
		return 0, fmt.Errorf("threshold must be >= 0")
	}
	return s.inventoryRepo.BulkSetThreshold(in.Threshold, in.CategoryID, in.ProductIDs)
}

func StockStatus(product *models.Product) string {
	return product.StockStatus()
}

func ProductInStock(product *models.Product) bool {
	return product.InStock()
}
