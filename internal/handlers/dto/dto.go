package dto

import (
	"auto-store-api/internal/models"
	"encoding/json"
	"fmt"
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
	ID              uuid.UUID                  `json:"id"`
	Email           string                     `json:"email"`
	FirstName       string                     `json:"first_name"`
	LastName        string                     `json:"last_name"`
	Role            string                     `json:"role"`
	Phone           string                     `json:"phone"`
	EmailVerified   bool                       `json:"email_verified"`
	CreatedAt       time.Time                  `json:"created_at"`
	MechanicProfile *MechanicProfileSummary    `json:"mechanic_profile,omitempty"`
}

type MechanicProfileSummary struct {
	ID           uuid.UUID `json:"id"`
	Status       string    `json:"status"`
	BusinessName string    `json:"business_name"`
	IsVerified   bool      `json:"is_verified"`
}

func UserToResponse(u *models.User) UserResponse {
	resp := UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Role:          string(u.Role),
		Phone:         u.Phone,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
	}
	if u.MechanicProfile != nil {
		resp.MechanicProfile = &MechanicProfileSummary{
			ID:           u.MechanicProfile.ID,
			Status:       string(u.MechanicProfile.Status),
			BusinessName: u.MechanicProfile.BusinessName,
			IsVerified:   u.MechanicProfile.IsVerified(),
		}
	}
	return resp
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
	// Images replaces all product images when set (JSON key present).
	// Omit the key to leave images unchanged. Use [] to remove all images.
	// Same shape as POST /products/:id/images items: url, alt_text, display_order, is_primary.
	Images *UpdateProductImages `json:"images"`
}

// UpdateProductImages accepts either:
// 1) []ProductImageItem objects, or
// 2) []string URLs (shorthand; mapped to ProductImageItem{URL: ...}).
type UpdateProductImages []ProductImageItem

func (u *UpdateProductImages) UnmarshalJSON(data []byte) error {
	var objs []ProductImageItem
	if err := json.Unmarshal(data, &objs); err == nil {
		*u = objs
		return nil
	}
	var urls []string
	if err := json.Unmarshal(data, &urls); err == nil {
		items := make([]ProductImageItem, len(urls))
		for i, v := range urls {
			items[i] = ProductImageItem{URL: v}
		}
		*u = items
		return nil
	}
	return fmt.Errorf("images must be an array of objects or array of url strings")
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
	ProductID      uuid.UUID `json:"product_id"`
	ProductIDCamel uuid.UUID `json:"productId"` // alias for camelCase JSON clients
	Quantity       int       `json:"quantity" binding:"required,min=1"`
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
	// Camel-case aliases for typical frontend payloads
	FirstNameCamel *string `json:"firstName"`
	LastNameCamel  *string `json:"lastName"`
}

// User role (Admin only; values stored in caps: ADMIN, VENDOR, CUSTOMER, MECHANIC)
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=ADMIN VENDOR CUSTOMER MECHANIC"`
}

// Mechanics
type MechanicApplyRequest struct {
	BusinessName    string                    `json:"business_name" binding:"required,max=200"`
	Bio             string                    `json:"bio" binding:"max=2000"`
	Phone           string                    `json:"phone" binding:"omitempty,phone"`
	Street          string                    `json:"street" binding:"max=200"`
	City            string                    `json:"city" binding:"required,max=100"`
	State           string                    `json:"state" binding:"required,max=100"`
	PostalCode      string                    `json:"postal_code" binding:"required,max=20"`
	Country         string                    `json:"country" binding:"omitempty,max=100"`
	Latitude        *float64                  `json:"latitude"`
	Longitude       *float64                  `json:"longitude"`
	ServiceRadiusKm float64                   `json:"service_radius_km" binding:"omitempty,gte=1,lte=500"`
	Documents       []MechanicDocumentRequest `json:"documents" binding:"omitempty,dive"`
}

type MechanicUpdateProfileRequest struct {
	BusinessName    *string  `json:"business_name" binding:"omitempty,max=200"`
	Bio             *string  `json:"bio" binding:"omitempty,max=2000"`
	Phone           *string  `json:"phone" binding:"omitempty,phone"`
	Street          *string  `json:"street" binding:"omitempty,max=200"`
	City            *string  `json:"city" binding:"omitempty,max=100"`
	State           *string  `json:"state" binding:"omitempty,max=100"`
	PostalCode      *string  `json:"postal_code" binding:"omitempty,max=20"`
	Country         *string  `json:"country" binding:"omitempty,max=100"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	ServiceRadiusKm *float64 `json:"service_radius_km" binding:"omitempty,gte=1,lte=500"`
}

type MechanicDocumentRequest struct {
	DocumentType string `json:"document_type" binding:"required,oneof=license insurance certification other"`
	URL          string `json:"url" binding:"required,url"`
	FileName     string `json:"file_name" binding:"max=255"`
}

type MechanicAdminActionRequest struct {
	Reason string `json:"reason"`
}

type MechanicDocumentResponse struct {
	ID           uuid.UUID `json:"id"`
	DocumentType string    `json:"document_type"`
	URL          string    `json:"url"`
	FileName     string    `json:"file_name"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type MechanicProfileResponse struct {
	ID              uuid.UUID                  `json:"id"`
	UserID          uuid.UUID                  `json:"user_id"`
	BusinessName    string                     `json:"business_name"`
	Bio             string                     `json:"bio"`
	Phone           string                     `json:"phone"`
	Street          string                     `json:"street"`
	City            string                     `json:"city"`
	State           string                     `json:"state"`
	PostalCode      string                     `json:"postal_code"`
	Country         string                     `json:"country"`
	Latitude        *float64                   `json:"latitude,omitempty"`
	Longitude       *float64                   `json:"longitude,omitempty"`
	ServiceRadiusKm float64                    `json:"service_radius_km"`
	Status          string                     `json:"status"`
	RatingAvg       float64                    `json:"rating_avg"`
	RatingCount     int                        `json:"rating_count"`
	VerifiedAt      *time.Time                 `json:"verified_at,omitempty"`
	Documents       []MechanicDocumentResponse `json:"documents,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
}

func MechanicDocumentToResponse(d *models.MechanicDocument) MechanicDocumentResponse {
	return MechanicDocumentResponse{
		ID:           d.ID,
		DocumentType: string(d.DocumentType),
		URL:          d.URL,
		FileName:     d.FileName,
		Status:       string(d.Status),
		CreatedAt:    d.CreatedAt,
	}
}

func MechanicProfileToResponse(p *models.MechanicProfile) MechanicProfileResponse {
	return mechanicProfileResponse(p, true)
}

func MechanicProfileToPublicResponse(p *models.MechanicProfile) MechanicProfileResponse {
	return mechanicProfileResponse(p, false)
}

func mechanicProfileResponse(p *models.MechanicProfile, includePrivate bool) MechanicProfileResponse {
	resp := MechanicProfileResponse{
		ID:              p.ID,
		UserID:          p.UserID,
		BusinessName:    p.BusinessName,
		Bio:             p.Bio,
		Phone:           p.Phone,
		Street:          p.Street,
		City:            p.City,
		State:           p.State,
		PostalCode:      p.PostalCode,
		Country:         p.Country,
		Latitude:        p.Latitude,
		Longitude:       p.Longitude,
		ServiceRadiusKm: p.ServiceRadiusKm,
		Status:          string(p.Status),
		RatingAvg:       p.RatingAvg,
		RatingCount:     p.RatingCount,
		VerifiedAt:      p.VerifiedAt,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
	if includePrivate && len(p.Documents) > 0 {
		resp.Documents = make([]MechanicDocumentResponse, len(p.Documents))
		for i, d := range p.Documents {
			resp.Documents[i] = MechanicDocumentToResponse(&d)
		}
	}
	return resp
}

// Notifications
type NotificationResponse struct {
	ID        uuid.UUID              `json:"id"`
	Type      string                 `json:"type"`
	Channel   string                 `json:"channel"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	ReadAt    *time.Time             `json:"read_at,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

type NotificationPreferenceResponse struct {
	EmailEnabled bool `json:"email_enabled"`
	SmsEnabled   bool `json:"sms_enabled"`
	PushEnabled  bool `json:"push_enabled"`
	InAppEnabled bool `json:"in_app_enabled"`
}

type UpdateNotificationPreferenceRequest struct {
	EmailEnabled *bool `json:"email_enabled"`
	SmsEnabled   *bool `json:"sms_enabled"`
	PushEnabled  *bool `json:"push_enabled"`
	InAppEnabled *bool `json:"in_app_enabled"`
}

func NotificationToResponse(n *models.Notification) NotificationResponse {
	resp := NotificationResponse{
		ID:        n.ID,
		Type:      string(n.Type),
		Channel:   string(n.Channel),
		Title:     n.Title,
		Body:      n.Body,
		ReadAt:    n.ReadAt,
		CreatedAt: n.CreatedAt,
	}
	if n.Payload != "" {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(n.Payload), &payload); err == nil {
			resp.Payload = payload
		}
	}
	return resp
}

func NotificationPreferenceToResponse(p *models.NotificationPreference) NotificationPreferenceResponse {
	return NotificationPreferenceResponse{
		EmailEnabled: p.EmailEnabled,
		SmsEnabled:   p.SmsEnabled,
		PushEnabled:  p.PushEnabled,
		InAppEnabled: p.InAppEnabled,
	}
}

// Installation marketplace
type InstallationJobTypeResponse struct {
	ID               uuid.UUID `json:"id"`
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	BaseLaborMinutes int       `json:"base_labor_minutes"`
	BaseLaborPrice   float64   `json:"base_labor_price"`
}

func InstallationJobTypeToResponse(j *models.InstallationJobType) InstallationJobTypeResponse {
	return InstallationJobTypeResponse{
		ID:               j.ID,
		Code:             j.Code,
		Name:             j.Name,
		Description:      j.Description,
		BaseLaborMinutes: j.BaseLaborMinutes,
		BaseLaborPrice:   j.BaseLaborPrice,
	}
}

type CreateInstallationQuoteRequest struct {
	OrderID           *uuid.UUID  `json:"order_id"`
	ProductIDs        []uuid.UUID `json:"product_ids"`
	VehicleMake       string      `json:"vehicle_make" binding:"required,max=100"`
	VehicleModel      string      `json:"vehicle_model" binding:"required,max=100"`
	VehicleYear       int         `json:"vehicle_year" binding:"required,gte=1900,lte=2100"`
	ServiceStreet     string      `json:"service_street" binding:"max=200"`
	ServiceCity       string      `json:"service_city" binding:"required,max=100"`
	ServiceState      string      `json:"service_state" binding:"required,max=100"`
	ServicePostalCode string      `json:"service_postal_code" binding:"required,max=20"`
	ServiceCountry    string      `json:"service_country" binding:"omitempty,max=100"`
	Latitude          *float64    `json:"latitude" binding:"required"`
	Longitude         *float64    `json:"longitude" binding:"required"`
	Notes             string      `json:"notes" binding:"max=2000"`
	SearchRadiusKm    float64     `json:"search_radius_km" binding:"omitempty,gte=1,lte=200"`
}

type InstallationQuoteLineResponse struct {
	ID                uuid.UUID `json:"id"`
	MechanicProfileID uuid.UUID `json:"mechanic_profile_id"`
	MechanicName      string    `json:"mechanic_name"`
	JobTypeID         uuid.UUID `json:"job_type_id"`
	JobTypeName       string    `json:"job_type_name"`
	LaborPrice        float64   `json:"labor_price"`
	EstimatedHours    float64   `json:"estimated_hours"`
	MechanicMessage   string    `json:"mechanic_message"`
	DistanceKm        float64   `json:"distance_km"`
	Status            string    `json:"status"`
	RatingAvg         float64   `json:"rating_avg,omitempty"`
}

type InstallationQuoteResponse struct {
	ID                uuid.UUID                       `json:"id"`
	Status            string                          `json:"status"`
	VehicleMake       string                          `json:"vehicle_make"`
	VehicleModel      string                          `json:"vehicle_model"`
	VehicleYear       int                             `json:"vehicle_year"`
	ServiceStreet     string                          `json:"service_street"`
	ServiceCity       string                          `json:"service_city"`
	ServiceState      string                          `json:"service_state"`
	ServicePostalCode string                          `json:"service_postal_code"`
	ServiceCountry    string                          `json:"service_country"`
	Latitude          *float64                        `json:"latitude,omitempty"`
	Longitude         *float64                        `json:"longitude,omitempty"`
	Notes             string                          `json:"notes"`
	ExpiresAt         time.Time                       `json:"expires_at"`
	Lines             []InstallationQuoteLineResponse   `json:"lines"`
	CreatedAt         time.Time                       `json:"created_at"`
}

func InstallationQuoteToResponse(q *models.InstallationQuote) InstallationQuoteResponse {
	resp := InstallationQuoteResponse{
		ID:                q.ID,
		Status:            string(q.Status),
		VehicleMake:       q.VehicleMake,
		VehicleModel:      q.VehicleModel,
		VehicleYear:       q.VehicleYear,
		ServiceStreet:     q.ServiceStreet,
		ServiceCity:       q.ServiceCity,
		ServiceState:      q.ServiceState,
		ServicePostalCode: q.ServicePostalCode,
		ServiceCountry:    q.ServiceCountry,
		Latitude:          q.Latitude,
		Longitude:         q.Longitude,
		Notes:             q.Notes,
		ExpiresAt:         q.ExpiresAt,
		CreatedAt:         q.CreatedAt,
	}
	if len(q.Lines) > 0 {
		resp.Lines = make([]InstallationQuoteLineResponse, len(q.Lines))
		for i, line := range q.Lines {
			name := ""
			rating := 0.0
			if line.MechanicProfile.ID != uuid.Nil {
				name = line.MechanicProfile.BusinessName
				rating = line.MechanicProfile.RatingAvg
			}
			jobName := ""
			if line.JobType.ID != uuid.Nil {
				jobName = line.JobType.Name
			}
			resp.Lines[i] = InstallationQuoteLineResponse{
				ID:                line.ID,
				MechanicProfileID: line.MechanicProfileID,
				MechanicName:      name,
				JobTypeID:         line.JobTypeID,
				JobTypeName:       jobName,
				LaborPrice:        line.LaborPrice,
				EstimatedHours:    line.EstimatedHours,
				MechanicMessage:   line.MechanicMessage,
				DistanceKm:        line.DistanceKm,
				Status:            string(line.Status),
				RatingAvg:         rating,
			}
		}
	}
	return resp
}

type CreateInstallationBookingRequest struct {
	QuoteID     uuid.UUID `json:"quote_id" binding:"required"`
	QuoteLineID uuid.UUID `json:"quote_line_id" binding:"required"`
	ScheduledAt time.Time `json:"scheduled_at" binding:"required"`
}

type CancelInstallationBookingRequest struct {
	Reason string `json:"reason" binding:"max=500"`
}

type InstallationBookingResponse struct {
	ID                uuid.UUID  `json:"id"`
	QuoteID           uuid.UUID  `json:"quote_id"`
	Status            string     `json:"status"`
	ScheduledAt       time.Time  `json:"scheduled_at"`
	MechanicProfileID uuid.UUID  `json:"mechanic_profile_id"`
	MechanicName      string     `json:"mechanic_name"`
	ServiceStreet     string     `json:"service_street"`
	ServiceCity       string     `json:"service_city"`
	ServiceState      string     `json:"service_state"`
	ServicePostalCode string     `json:"service_postal_code"`
	LaborTotal        float64    `json:"labor_total"`
	PartsTotal        float64    `json:"parts_total"`
	PlatformFee       float64    `json:"platform_fee"`
	TotalAmount       float64    `json:"total_amount"`
	PaymentStatus     string     `json:"payment_status"`
	CreatedAt         time.Time  `json:"created_at"`
}

func InstallationBookingToResponse(b *models.InstallationBooking) InstallationBookingResponse {
	name := ""
	if b.MechanicProfile.ID != uuid.Nil {
		name = b.MechanicProfile.BusinessName
	}
	return InstallationBookingResponse{
		ID:                b.ID,
		QuoteID:           b.QuoteID,
		Status:            string(b.Status),
		ScheduledAt:       b.ScheduledAt,
		MechanicProfileID: b.MechanicProfileID,
		MechanicName:      name,
		ServiceStreet:     b.ServiceStreet,
		ServiceCity:       b.ServiceCity,
		ServiceState:      b.ServiceState,
		ServicePostalCode: b.ServicePostalCode,
		LaborTotal:        b.LaborTotal,
		PartsTotal:        b.PartsTotal,
		PlatformFee:       b.PlatformFee,
		TotalAmount:       b.TotalAmount,
		PaymentStatus:     string(b.PaymentStatus),
		CreatedAt:         b.CreatedAt,
	}
}

type RespondInstallationQuoteLineRequest struct {
	LaborPrice      *float64 `json:"labor_price" binding:"omitempty,gte=0"`
	EstimatedHours  *float64 `json:"estimated_hours" binding:"omitempty,gte=0.25,lte=24"`
	MechanicMessage *string  `json:"mechanic_message" binding:"omitempty,max=1000"`
}

type UpdateInstallationBookingStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=confirmed en_route in_progress completed cancelled"`
}

type MechanicInstallServicesRequest struct {
	JobTypeIDs []uuid.UUID `json:"job_type_ids" binding:"required,min=1,dive"`
}
