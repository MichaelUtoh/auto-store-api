package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QuoteStatus string

const (
	QuoteStatusReady     QuoteStatus = "ready"
	QuoteStatusExpired   QuoteStatus = "expired"
	QuoteStatusBooked    QuoteStatus = "booked"
	QuoteStatusCancelled QuoteStatus = "cancelled"
)

type QuoteLineStatus string

const (
	QuoteLineStatusOffered  QuoteLineStatus = "offered"
	QuoteLineStatusSelected QuoteLineStatus = "selected"
	QuoteLineStatusDeclined QuoteLineStatus = "declined"
)

type BookingStatus string

const (
	BookingStatusPendingPayment BookingStatus = "pending_payment"
	BookingStatusConfirmed      BookingStatus = "confirmed"
	BookingStatusEnRoute        BookingStatus = "en_route"
	BookingStatusInProgress     BookingStatus = "in_progress"
	BookingStatusCompleted      BookingStatus = "completed"
	BookingStatusCancelled      BookingStatus = "cancelled"
)

func IsValidBookingStatus(s BookingStatus) bool {
	switch s {
	case BookingStatusPendingPayment, BookingStatusConfirmed, BookingStatusEnRoute,
		BookingStatusInProgress, BookingStatusCompleted, BookingStatusCancelled:
		return true
	default:
		return false
	}
}

// InstallationJobType is the canonical catalog of installable jobs (e.g. brake pad replacement).
type InstallationJobType struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Code               string         `gorm:"uniqueIndex;not null" json:"code"`
	Name               string         `gorm:"not null" json:"name"`
	Description        string         `gorm:"type:text" json:"description"`
	BaseLaborMinutes   int            `gorm:"column:base_labor_minutes;default:60" json:"base_labor_minutes"`
	BaseLaborPrice     float64        `gorm:"column:base_labor_price;type:decimal(12,2);not null" json:"base_labor_price"`
	CategoryID         *uuid.UUID     `gorm:"type:uuid;column:category_id" json:"category_id,omitempty"`
	IsActive           bool           `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (InstallationJobType) TableName() string { return "installation_job_types" }

func (j *InstallationJobType) BeforeCreate(tx *gorm.DB) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	return nil
}

// MechanicInstallService links a verified mechanic to job types they perform.
type MechanicInstallService struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	MechanicProfileID uuid.UUID `gorm:"type:uuid;not null;index" json:"mechanic_profile_id"`
	JobTypeID         uuid.UUID `gorm:"type:uuid;not null;index" json:"job_type_id"`
	LaborPrice        float64   `gorm:"column:labor_price;type:decimal(12,2)" json:"labor_price"`
	IsActive          bool      `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	MechanicProfile MechanicProfile       `gorm:"foreignKey:MechanicProfileID" json:"mechanic_profile,omitempty"`
	JobType         InstallationJobType   `gorm:"foreignKey:JobTypeID" json:"job_type,omitempty"`
}

func (MechanicInstallService) TableName() string { return "mechanic_install_services" }

func (m *MechanicInstallService) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// InstallationQuote is a customer's request for installation quotes from nearby mechanics.
type InstallationQuote struct {
	ID                uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	UserID            uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	OrderID           *uuid.UUID     `gorm:"type:uuid;index" json:"order_id,omitempty"`
	Status            QuoteStatus    `gorm:"type:varchar(20);not null;default:'ready';index" json:"status"`
	VehicleMake       string         `gorm:"column:vehicle_make;not null" json:"vehicle_make"`
	VehicleModel      string         `gorm:"column:vehicle_model;not null" json:"vehicle_model"`
	VehicleYear       int            `gorm:"column:vehicle_year;not null" json:"vehicle_year"`
	ServiceStreet     string         `gorm:"column:service_street" json:"service_street"`
	ServiceCity       string         `gorm:"column:service_city;not null" json:"service_city"`
	ServiceState      string         `gorm:"column:service_state;not null" json:"service_state"`
	ServicePostalCode string         `gorm:"column:service_postal_code;not null" json:"service_postal_code"`
	ServiceCountry    string         `gorm:"column:service_country;default:'US'" json:"service_country"`
	Latitude          *float64       `gorm:"type:decimal(10,7)" json:"latitude,omitempty"`
	Longitude         *float64       `gorm:"type:decimal(10,7)" json:"longitude,omitempty"`
	Notes             string         `gorm:"type:text" json:"notes"`
	ExpiresAt         time.Time      `gorm:"column:expires_at;not null;index" json:"expires_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	User  User                    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Order *Order                  `gorm:"foreignKey:OrderID" json:"order,omitempty"`
	Lines []InstallationQuoteLine `gorm:"foreignKey:QuoteID" json:"lines,omitempty"`
	Items []InstallationQuoteItem `gorm:"foreignKey:QuoteID" json:"items,omitempty"`
}

func (InstallationQuote) TableName() string { return "installation_quotes" }

func (q *InstallationQuote) BeforeCreate(tx *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	if q.ServiceCountry == "" {
		q.ServiceCountry = "US"
	}
	return nil
}

// InstallationQuoteItem snapshots products needing installation on this quote.
type InstallationQuoteItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	QuoteID   uuid.UUID `gorm:"type:uuid;not null;index" json:"quote_id"`
	ProductID uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	JobTypeID uuid.UUID `gorm:"type:uuid;not null;index" json:"job_type_id"`
	Quantity  int       `gorm:"not null;default:1" json:"quantity"`
	CreatedAt time.Time `json:"created_at"`

	Product Product             `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	JobType InstallationJobType `gorm:"foreignKey:JobTypeID" json:"job_type,omitempty"`
}

func (InstallationQuoteItem) TableName() string { return "installation_quote_items" }

func (i *InstallationQuoteItem) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

// InstallationQuoteLine is a per-mechanic price offer for a quote.
type InstallationQuoteLine struct {
	ID                uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	QuoteID           uuid.UUID       `gorm:"type:uuid;not null;index" json:"quote_id"`
	MechanicProfileID uuid.UUID       `gorm:"type:uuid;not null;index" json:"mechanic_profile_id"`
	JobTypeID         uuid.UUID       `gorm:"type:uuid;not null;index" json:"job_type_id"`
	LaborPrice        float64         `gorm:"column:labor_price;type:decimal(12,2);not null" json:"labor_price"`
	EstimatedHours    float64         `gorm:"column:estimated_hours;type:decimal(6,2);not null" json:"estimated_hours"`
	MechanicMessage   string          `gorm:"column:mechanic_message;type:text" json:"mechanic_message"`
	DistanceKm        float64         `gorm:"column:distance_km;type:decimal(8,2)" json:"distance_km"`
	Status            QuoteLineStatus `gorm:"type:varchar(20);not null;default:'offered'" json:"status"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`

	MechanicProfile MechanicProfile       `gorm:"foreignKey:MechanicProfileID" json:"mechanic_profile,omitempty"`
	JobType         InstallationJobType   `gorm:"foreignKey:JobTypeID" json:"job_type,omitempty"`
}

func (InstallationQuoteLine) TableName() string { return "installation_quote_lines" }

func (l *InstallationQuoteLine) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

// InstallationBooking is a confirmed installation appointment.
type InstallationBooking struct {
	ID                uuid.UUID     `gorm:"type:uuid;primary_key" json:"id"`
	QuoteID           uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex" json:"quote_id"`
	QuoteLineID       uuid.UUID     `gorm:"type:uuid;not null" json:"quote_line_id"`
	UserID            uuid.UUID     `gorm:"type:uuid;not null;index" json:"user_id"`
	MechanicProfileID uuid.UUID     `gorm:"type:uuid;not null;index" json:"mechanic_profile_id"`
	MechanicUserID    uuid.UUID     `gorm:"type:uuid;not null;index" json:"mechanic_user_id"`
	Status            BookingStatus `gorm:"type:varchar(30);not null;default:'pending_payment';index" json:"status"`
	ScheduledAt       time.Time     `gorm:"column:scheduled_at;not null;index" json:"scheduled_at"`
	ServiceStreet     string        `gorm:"column:service_street" json:"service_street"`
	ServiceCity       string        `gorm:"column:service_city;not null" json:"service_city"`
	ServiceState      string        `gorm:"column:service_state;not null" json:"service_state"`
	ServicePostalCode string        `gorm:"column:service_postal_code;not null" json:"service_postal_code"`
	ServiceCountry    string        `gorm:"column:service_country;default:'US'" json:"service_country"`
	LaborTotal        float64       `gorm:"column:labor_total;type:decimal(12,2);not null" json:"labor_total"`
	PartsTotal        float64       `gorm:"column:parts_total;type:decimal(12,2);default:0" json:"parts_total"`
	PlatformFee       float64       `gorm:"column:platform_fee;type:decimal(12,2);default:0" json:"platform_fee"`
	TotalAmount       float64       `gorm:"column:total_amount;type:decimal(12,2);not null" json:"total_amount"`
	PaymentStatus     PaymentStatus `gorm:"column:payment_status;type:varchar(20);default:'pending'" json:"payment_status"`
	CancellationReason string       `gorm:"column:cancellation_reason;type:text" json:"cancellation_reason,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	Quote           InstallationQuote       `gorm:"foreignKey:QuoteID" json:"quote,omitempty"`
	QuoteLine       InstallationQuoteLine   `gorm:"foreignKey:QuoteLineID" json:"quote_line,omitempty"`
	MechanicProfile MechanicProfile         `gorm:"foreignKey:MechanicProfileID" json:"mechanic_profile,omitempty"`
	Payment         *BookingPayment         `gorm:"foreignKey:BookingID" json:"payment,omitempty"`
}

func (InstallationBooking) TableName() string { return "installation_bookings" }

func (b *InstallationBooking) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.ServiceCountry == "" {
		b.ServiceCountry = "US"
	}
	return nil
}

// BookingPayment stores external payment provider references.
// Planned provider: Paystack (reference/transaction id in PaymentIntentID until a dedicated column is added).
type BookingPayment struct {
	ID              uuid.UUID     `gorm:"type:uuid;primary_key" json:"id"`
	BookingID       uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex" json:"booking_id"`
	Provider        string        `gorm:"not null;default:'manual'" json:"provider"` // manual | paystack
	PaymentIntentID string        `gorm:"column:payment_intent_id" json:"payment_intent_id"` // Paystack reference when integrated
	Amount          float64       `gorm:"type:decimal(12,2);not null" json:"amount"`
	Status          PaymentStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

func (BookingPayment) TableName() string { return "booking_payments" }

func (p *BookingPayment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
