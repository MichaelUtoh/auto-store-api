package services

import (
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrEmptyCart       = errors.New("cart is empty")
	ErrAddressNotFound = errors.New("address not found")
	ErrOrderNotFound   = errors.New("order not found")
)

type OrderService struct {
	orderRepo   *repositories.OrderRepository
	cartRepo    *repositories.CartRepository
	addressRepo *repositories.AddressRepository
	productRepo *repositories.ProductRepository
	db          *gorm.DB
}

func NewOrderService(
	orderRepo *repositories.OrderRepository,
	cartRepo *repositories.CartRepository,
	addressRepo *repositories.AddressRepository,
	productRepo *repositories.ProductRepository,
	db *gorm.DB,
) *OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		cartRepo:    cartRepo,
		addressRepo: addressRepo,
		productRepo: productRepo,
		db:          db,
	}
}

func (s *OrderService) Create(userID, shippingAddrID, billingAddrID uuid.UUID, paymentMethod string) (*models.Order, error) {
	cartItems, err := s.cartRepo.GetByUserID(userID)
	if err != nil || len(cartItems) == 0 {
		return nil, ErrEmptyCart
	}
	shipAddr, err := s.addressRepo.GetByIDAndUser(shippingAddrID, userID)
	if err != nil || shipAddr == nil {
		return nil, ErrAddressNotFound
	}
	billAddr, err := s.addressRepo.GetByIDAndUser(billingAddrID, userID)
	if err != nil || billAddr == nil {
		return nil, ErrAddressNotFound
	}

	var subtotal float64
	var orderItems []models.OrderItem
	for _, ci := range cartItems {
		product, _ := s.productRepo.GetByID(ci.ProductID)
		if product == nil || product.StockQuantity < ci.Quantity {
			continue
		}
		unitPrice := product.Price
		totalPrice := unitPrice * float64(ci.Quantity)
		subtotal += totalPrice
		orderItems = append(orderItems, models.OrderItem{
			ProductID:  ci.ProductID,
			Quantity:   ci.Quantity,
			UnitPrice:  unitPrice,
			TotalPrice: totalPrice,
		})
	}
	if len(orderItems) == 0 {
		return nil, ErrEmptyCart
	}

	tax := 0.0
	shippingCost := 0.0
	total := subtotal + tax + shippingCost

	order := &models.Order{
		UserID:            userID,
		Status:            models.OrderStatusPending,
		Subtotal:          subtotal,
		Tax:               tax,
		ShippingCost:      shippingCost,
		Total:             total,
		ShippingAddressID: shippingAddrID,
		BillingAddressID:  billingAddrID,
		PaymentMethod:    paymentMethod,
		PaymentStatus:     models.PaymentPending,
	}
	if err := s.orderRepo.Create(order); err != nil {
		return nil, err
	}
	for i := range orderItems {
		orderItems[i].OrderID = order.ID
		if err := s.orderRepo.AddItem(&orderItems[i]); err != nil {
			return nil, err
		}
		s.db.Model(&models.Product{}).Where("id = ?", orderItems[i].ProductID).
			Update("stock_quantity", gorm.Expr("stock_quantity - ?", orderItems[i].Quantity))
	}
	_ = s.cartRepo.Clear(userID)
	return order, nil
}

func (s *OrderService) GetByID(id, userID uuid.UUID) (*models.Order, error) {
	o, err := s.orderRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if o.UserID != userID {
		return nil, ErrOrderNotFound
	}
	return o, nil
}

func (s *OrderService) ListByUser(userID uuid.UUID, page, limit int) ([]models.Order, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.orderRepo.GetByUserID(userID, offset, limit)
}

func (s *OrderService) Cancel(id, userID uuid.UUID) error {
	o, err := s.orderRepo.GetByID(id)
	if err != nil {
		return err
	}
	if o.UserID != userID {
		return ErrOrderNotFound
	}
	if o.Status != models.OrderStatusPending {
		return errors.New("only pending orders can be cancelled")
	}
	o.Status = models.OrderStatusCancelled
	return s.orderRepo.Update(o)
}

func (s *OrderService) ListAll(page, limit int, status string) ([]models.Order, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.orderRepo.ListAll(offset, limit, status)
}

func (s *OrderService) UpdateStatus(id uuid.UUID, status models.OrderStatus) (*models.Order, error) {
	o, err := s.orderRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	o.Status = status
	if err := s.orderRepo.Update(o); err != nil {
		return nil, err
	}
	return s.orderRepo.GetByID(id)
}
