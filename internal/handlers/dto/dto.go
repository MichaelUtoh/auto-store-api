package dto

import (
	"auto-store-api/internal/models"
	"time"

	"github.com/google/uuid"
)

// Auth
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"max=100"`
	LastName  string `json:"last_name" binding:"max=100"`
	Phone     string `json:"phone" binding:"omitempty,phone"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
	User         UserResponse `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type UserResponse struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Role          string    `json:"role"`
	Phone         string    `json:"phone"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

func UserToResponse(u *models.User) UserResponse {
	return UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Role:          string(u.Role),
		Phone:         u.Phone,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
	}
}

// Products
type CreateProductRequest struct {
	SKU                string      `json:"sku" binding:"required,max=100"`
	Name               string      `json:"name" binding:"required,max=255"`
	Description        string      `json:"description"`
	Brand              string      `json:"brand" binding:"max=100"`
	ManufacturerPartNo string      `json:"manufacturer_part_number" binding:"max=100"`
	Price              float64     `json:"price" binding:"required,gt=0"`
	CostPrice          float64     `json:"cost_price" binding:"gte=0"`
	StockQuantity      int         `json:"stock_quantity" binding:"gte=0"`
	Weight             float64     `json:"weight" binding:"gte=0"`
	Dimensions         string      `json:"dimensions" binding:"max=100"`
	Condition          string      `json:"condition" binding:"omitempty,oneof=new refurbished used"`
	WarrantyMonths     int         `json:"warranty_months" binding:"gte=0"`
	CategoryIDs        []uuid.UUID `json:"category_ids"`
	TagIDs             []uuid.UUID `json:"tag_ids"`
}

// CreateProductsBatchRequest is the request body for batch product creation.
type CreateProductsBatchRequest struct {
	Products []CreateProductRequest `json:"products" binding:"required,dive"`
}

// BatchProductResult represents a single created product in a batch response.
type BatchProductResult struct {
	Index    int           `json:"index"`
	Product  *models.Product `json:"product"`
}

// BatchProductError represents a failed product in a batch response.
type BatchProductError struct {
	Index   int    `json:"index"`
	SKU     string `json:"sku"`
	Message string `json:"message"`
}

// CreateProductsBatchResponse is the response for batch product creation.
type CreateProductsBatchResponse struct {
	Created []BatchProductResult `json:"created"`
	Failed  []BatchProductError  `json:"failed"`
}

type UpdateProductRequest struct {
	Name               *string     `json:"name"`
	Description        *string     `json:"description"`
	Brand              *string     `json:"brand"`
	ManufacturerPartNo *string     `json:"manufacturer_part_number"`
	Price              *float64    `json:"price"`
	CostPrice          *float64    `json:"cost_price"`
	StockQuantity      *int        `json:"stock_quantity"`
	Weight             *float64    `json:"weight"`
	Dimensions         *string     `json:"dimensions"`
	Condition          *string     `json:"condition"`
	WarrantyMonths     *int        `json:"warranty_months"`
	CategoryIDs        []uuid.UUID `json:"category_ids"`
	TagIDs             []uuid.UUID `json:"tag_ids"`
}

type SearchProductsQuery struct {
	Q         string   `form:"q"`
	Category  string   `form:"category"`
	Tags      []string `form:"tags"`
	Make      string   `form:"make"`
	Model     string   `form:"model"`
	Year      string   `form:"year"`
	MinPrice  *float64 `form:"minPrice"`
	MaxPrice  *float64 `form:"maxPrice"`
	Condition string   `form:"condition"`
	Brand     string   `form:"brand"`
	Sort      string   `form:"sort"`
	Page      int      `form:"page"`
	Limit     int      `form:"limit"`
}

// AddProductImagesRequest is the body for adding images to a product.
type AddProductImagesRequest struct {
	Images []ProductImageItem `json:"images" binding:"required,min=1,dive"`
}

// AddVehicleCompatibilitiesRequest is the body for adding vehicle compatibilities to a product.
type AddVehicleCompatibilitiesRequest struct {
	Compatibilities []VehicleCompatibilityItem `json:"compatibilities" binding:"required,min=1,dive"`
}

// VehicleCompatibilityItem is a single vehicle fit (make, model, year range, etc.).
type VehicleCompatibilityItem struct {
	Make      string `json:"make" binding:"required,max=100"`
	Model     string `json:"model" binding:"required,max=100"`
	YearStart int    `json:"year_start"` // 0 = unspecified
	YearEnd   int    `json:"year_end"`   // 0 = unspecified
	Engine    string `json:"engine" binding:"max=100"`
	Trim      string `json:"trim" binding:"max=100"`
	Notes     string `json:"notes" binding:"max=500"`
}

// ProductImageItem is a single image in an add-images request.
type ProductImageItem struct {
	URL          string `json:"url" binding:"required,url"`
	AltText      string `json:"alt_text" binding:"max=255"`
	DisplayOrder int    `json:"display_order"`
	IsPrimary    bool   `json:"is_primary"`
}

// Categories
type CreateCategoryRequest struct {
	ParentID    *uuid.UUID `json:"parent_id"`
	Name        string     `json:"name" binding:"required,max=100"`
	Slug        string     `json:"slug" binding:"omitempty,slug"`
	Description string     `json:"description"`
}

type UpdateCategoryRequest struct {
	Name        *string    `json:"name"`
	Slug        *string    `json:"slug"`
	Description *string    `json:"description"`
	ParentID    *uuid.UUID `json:"parent_id"`
}

// Cart
type AddCartItemRequest struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Quantity  int       `json:"quantity" binding:"required,min=1"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1"`
}

// Orders
type CreateOrderRequest struct {
	ShippingAddressID uuid.UUID `json:"shipping_address_id" binding:"required"`
	BillingAddressID  uuid.UUID `json:"billing_address_id" binding:"required"`
	PaymentMethod     string    `json:"payment_method" binding:"required,max=50"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending confirmed processing shipped delivered cancelled"`
}

// Addresses
type CreateAddressRequest struct {
	Type       string `json:"type" binding:"required,oneof=shipping billing"`
	Street     string `json:"street" binding:"required,max=255"`
	City       string `json:"city" binding:"required,max=100"`
	State      string `json:"state" binding:"max=100"`
	PostalCode string `json:"postal_code" binding:"max=20"`
	Country    string `json:"country" binding:"required,max=100"`
	IsDefault  bool   `json:"is_default"`
}

type UpdateAddressRequest struct {
	Street     *string `json:"street"`
	City       *string `json:"city"`
	State      *string `json:"state"`
	PostalCode *string `json:"postal_code"`
	Country    *string `json:"country"`
	IsDefault  *bool   `json:"is_default"`
}

// Reviews
type CreateReviewRequest struct {
	Rating  int    `json:"rating" binding:"required,min=1,max=5"`
	Title   string `json:"title" binding:"max=200"`
	Comment string `json:"comment" binding:"max=2000"`
}

// User profile
type UpdateProfileRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Phone     *string `json:"phone"`
}

// User role (Admin only; values stored in caps: ADMIN, VENDOR, CUSTOMER)
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=ADMIN VENDOR CUSTOMER"`
}
