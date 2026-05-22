package services

import (
	"auto-store-api/internal/config"
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"auto-store-api/pkg/paystack"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrPaystackNotConfigured = errors.New("paystack is not configured")
	ErrPaymentNotPending       = errors.New("payment is not pending")
	ErrPaymentInvalidAmount    = errors.New("invalid payment amount")
	ErrPaymentReferenceUnknown = errors.New("unknown payment reference")
	ErrBookingSplitInvalid     = errors.New("booking split configuration is invalid")
)

const (
	paymentEntityOrder   = "order"
	paymentEntityBooking = "booking"
)

// PaymentInitializeResult is returned when starting a Paystack checkout.
type PaymentInitializeResult struct {
	AuthorizationURL string `json:"authorization_url"`
	AccessCode       string `json:"access_code"`
	Reference        string `json:"reference"`
	PublicKey        string `json:"public_key"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
}

type PaymentService struct {
	cfg          config.PaystackConfig
	client       *paystack.Client
	orderRepo    *repositories.OrderRepository
	installRepo  *repositories.InstallationRepository
	mechanicRepo *repositories.MechanicRepository
	userRepo     *repositories.UserRepository
	db           *gorm.DB
	log          *zap.Logger
}

func NewPaymentService(
	cfg config.PaystackConfig,
	orderRepo *repositories.OrderRepository,
	installRepo *repositories.InstallationRepository,
	mechanicRepo *repositories.MechanicRepository,
	userRepo *repositories.UserRepository,
	db *gorm.DB,
	log *zap.Logger,
) *PaymentService {
	var client *paystack.Client
	if cfg.Enabled {
		client = paystack.NewClient(cfg.SecretKey)
	}
	return &PaymentService{
		cfg:          cfg,
		client:       client,
		orderRepo:    orderRepo,
		installRepo:  installRepo,
		mechanicRepo: mechanicRepo,
		userRepo:     userRepo,
		db:           db,
		log:          log,
	}
}

func (s *PaymentService) requireClient() error {
	if s.client == nil || !s.cfg.Enabled {
		return ErrPaystackNotConfigured
	}
	return nil
}

func (s *PaymentService) PublicKey() string {
	return s.cfg.PublicKey
}

func (s *PaymentService) InitializeOrderPayment(userID, orderID uuid.UUID) (*PaymentInitializeResult, error) {
	if err := s.requireClient(); err != nil {
		return nil, err
	}
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.UserID != userID {
		return nil, ErrOrderNotFound
	}
	if order.PaymentStatus != models.PaymentPending {
		return nil, ErrPaymentNotPending
	}
	if order.Status == models.OrderStatusCancelled {
		return nil, errors.New("order is cancelled")
	}
	return s.initialize(userID, paymentEntityOrder, orderID, order.Total, nil, func(ref string) error {
		order.PaymentMethod = "paystack"
		order.PaymentReference = ref
		return s.orderRepo.Update(order)
	})
}

func (s *PaymentService) InitializeBookingPayment(userID, bookingID uuid.UUID) (*PaymentInitializeResult, error) {
	if err := s.requireClient(); err != nil {
		return nil, err
	}
	booking, err := s.installRepo.GetBookingByIDForUser(bookingID, userID)
	if err != nil {
		return nil, ErrInstallationBookingNotFound
	}
	if booking.PaymentStatus != models.PaymentPending {
		return nil, ErrPaymentNotPending
	}
	if booking.Status != models.BookingStatusPendingPayment {
		return nil, errors.New("booking is not awaiting payment")
	}
	split, err := s.bookingSplitParams(booking)
	if err != nil {
		return nil, err
	}
	return s.initialize(userID, paymentEntityBooking, bookingID, booking.TotalAmount, split, func(ref string) error {
		return s.db.Model(&models.BookingPayment{}).
			Where("booking_id = ?", bookingID).
			Updates(map[string]interface{}{
				"provider":          "paystack",
				"payment_intent_id": ref,
			}).Error
	})
}

// bookingSplitParams returns Paystack split settings: platform keeps platform_fee (flat),
// mechanic subaccount receives labor + parts.
func (s *PaymentService) bookingSplitParams(booking *models.InstallationBooking) (*paystack.InitializeParams, error) {
	if !s.cfg.SplitEnabled {
		return nil, nil
	}
	profile, err := s.mechanicRepo.GetProfileByID(booking.MechanicProfileID)
	if err != nil {
		return nil, ErrInstallationBookingNotFound
	}
	if profile.PaystackSubaccountCode == "" {
		if s.cfg.RequireSplitForBookings {
			return nil, ErrMechanicPayoutNotConfigured
		}
		s.log.Warn("booking payment without mechanic subaccount",
			zap.String("booking_id", booking.ID.String()),
			zap.String("mechanic_profile_id", profile.ID.String()),
		)
		return nil, nil
	}

	platformMinor, err := toMinorUnits(booking.PlatformFee, s.cfg.Currency)
	if err != nil {
		return nil, err
	}
	totalMinor, err := toMinorUnits(booking.TotalAmount, s.cfg.Currency)
	if err != nil {
		return nil, err
	}
	if platformMinor >= totalMinor {
		return nil, ErrBookingSplitInvalid
	}

	return &paystack.InitializeParams{
		Subaccount:        profile.PaystackSubaccountCode,
		TransactionCharge: platformMinor,
	}, nil
}

func (s *PaymentService) initialize(
	userID uuid.UUID,
	entityType string,
	entityID uuid.UUID,
	amount float64,
	split *paystack.InitializeParams,
	persistRef func(string) error,
) (*PaymentInitializeResult, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	amountMinor, err := toMinorUnits(amount, s.cfg.Currency)
	if err != nil {
		return nil, err
	}

	ref := fmt.Sprintf("%s-%s-%s", entityType, entityID.String()[:8], uuid.New().String()[:8])
	meta := map[string]string{
		"entity_type": entityType,
		"entity_id":   entityID.String(),
		"user_id":     userID.String(),
	}

	init := paystack.InitializeParams{
		Email:       user.Email,
		Amount:      amountMinor,
		Currency:    s.cfg.Currency,
		Reference:   ref,
		CallbackURL: s.cfg.CallbackURL,
		Metadata:    meta,
	}
	if split != nil {
		init.Subaccount = split.Subaccount
		init.TransactionCharge = split.TransactionCharge
		init.Bearer = split.Bearer
		if split.Subaccount != "" {
			meta["split"] = "mechanic_subaccount"
			meta["platform_fee_minor"] = fmt.Sprintf("%d", split.TransactionCharge)
		}
		init.Metadata = meta
	}

	resp, err := s.client.Initialize(init)
	if err != nil {
		return nil, err
	}
	if err := persistRef(ref); err != nil {
		return nil, err
	}

	return &PaymentInitializeResult{
		AuthorizationURL: resp.Data.AuthorizationURL,
		AccessCode:       resp.Data.AccessCode,
		Reference:        resp.Data.Reference,
		PublicKey:        s.cfg.PublicKey,
		Amount:           amountMinor,
		Currency:         s.cfg.Currency,
	}, nil
}

// VerifyPayment confirms a transaction with Paystack and updates the related order or booking.
func (s *PaymentService) VerifyPayment(reference string) (entityType string, entityID uuid.UUID, err error) {
	if err := s.requireClient(); err != nil {
		return "", uuid.Nil, err
	}
	if reference == "" {
		return "", uuid.Nil, errors.New("reference is required")
	}

	resp, err := s.client.Verify(reference)
	if err != nil {
		return "", uuid.Nil, err
	}
	if !isSuccessfulTransaction(resp.Data.Status) {
		return "", uuid.Nil, errors.New("payment not successful")
	}

	entityType = resp.Data.Metadata.EntityType
	entityIDStr := resp.Data.Metadata.EntityID
	if entityType == "" || entityIDStr == "" {
		entityType, entityID, err = s.resolveEntityFromReference(reference)
		if err != nil {
			return "", uuid.Nil, err
		}
	} else {
		entityID, err = uuid.Parse(entityIDStr)
		if err != nil {
			return "", uuid.Nil, err
		}
	}

	if err := s.markPaid(entityType, entityID, reference, resp.Data.Amount); err != nil {
		return "", uuid.Nil, err
	}
	return entityType, entityID, nil
}

// HandleWebhook processes a signed Paystack webhook payload.
func (s *PaymentService) HandleWebhook(body []byte, signature string) error {
	if s.cfg.WebhookSecret == "" {
		return ErrPaystackNotConfigured
	}
	if !paystack.VerifyWebhookSignature(s.cfg.WebhookSecret, body, signature) {
		return errors.New("invalid webhook signature")
	}

	var evt paystack.WebhookEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		return err
	}

	switch evt.Event {
	case "charge.success":
		var data paystack.ChargeWebhookData
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			return err
		}
		if !isSuccessfulTransaction(data.Status) {
			return nil
		}
		entityType := data.Metadata.EntityType
		entityIDStr := data.Metadata.EntityID
		var entityID uuid.UUID
		var err error
		if entityType == "" || entityIDStr == "" {
			entityType, entityID, err = s.resolveEntityFromReference(data.Reference)
			if err != nil {
				return err
			}
		} else {
			entityID, err = uuid.Parse(entityIDStr)
			if err != nil {
				return err
			}
		}
		return s.markPaid(entityType, entityID, data.Reference, data.Amount)

	case "refund.processed", "refund.failed":
		var data paystack.RefundWebhookData
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			return err
		}
		return s.handleRefundWebhook(data)

	default:
		return nil
	}
}

func (s *PaymentService) resolveEntityFromReference(ref string) (string, uuid.UUID, error) {
	if strings.HasPrefix(ref, paymentEntityOrder+"-") {
		order, err := s.orderRepo.GetByPaymentReference(ref)
		if err != nil {
			return "", uuid.Nil, ErrPaymentReferenceUnknown
		}
		return paymentEntityOrder, order.ID, nil
	}
	if strings.HasPrefix(ref, paymentEntityBooking+"-") {
		booking, err := s.installRepo.GetBookingByPaymentReference(ref)
		if err != nil {
			return "", uuid.Nil, ErrPaymentReferenceUnknown
		}
		return paymentEntityBooking, booking.ID, nil
	}
	return "", uuid.Nil, ErrPaymentReferenceUnknown
}

func (s *PaymentService) markPaid(entityType string, entityID uuid.UUID, reference string, paidMinor int64) error {
	switch entityType {
	case paymentEntityOrder:
		return s.markOrderPaid(entityID, reference, paidMinor)
	case paymentEntityBooking:
		return s.markBookingPaid(entityID, reference, paidMinor)
	default:
		return fmt.Errorf("unknown entity type: %s", entityType)
	}
}

func (s *PaymentService) markOrderPaid(orderID uuid.UUID, reference string, paidMinor int64) error {
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return ErrOrderNotFound
	}
	if order.PaymentStatus == models.PaymentPaid {
		return nil
	}
	expected, err := toMinorUnits(order.Total, s.cfg.Currency)
	if err != nil {
		return err
	}
	if paidMinor != expected {
		s.log.Warn("paystack amount mismatch for order",
			zap.String("order_id", orderID.String()),
			zap.Int64("expected", expected),
			zap.Int64("paid", paidMinor),
		)
	}

	order.PaymentStatus = models.PaymentPaid
	order.PaymentReference = reference
	order.PaymentMethod = "paystack"
	if order.Status == models.OrderStatusPending {
		order.Status = models.OrderStatusConfirmed
	}
	return s.orderRepo.Update(order)
}

func (s *PaymentService) markBookingPaid(bookingID uuid.UUID, reference string, paidMinor int64) error {
	booking, err := s.installRepo.GetBookingByID(bookingID)
	if err != nil {
		return ErrInstallationBookingNotFound
	}
	if booking.PaymentStatus == models.PaymentPaid {
		return nil
	}
	expected, err := toMinorUnits(booking.TotalAmount, s.cfg.Currency)
	if err != nil {
		return err
	}
	if paidMinor != expected {
		s.log.Warn("paystack amount mismatch for booking",
			zap.String("booking_id", bookingID.String()),
			zap.Int64("expected", expected),
			zap.Int64("paid", paidMinor),
		)
	}

	booking.PaymentStatus = models.PaymentPaid
	if booking.Status == models.BookingStatusPendingPayment {
		booking.Status = models.BookingStatusConfirmed
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(booking).Error; err != nil {
			return err
		}
		return tx.Model(&models.BookingPayment{}).
			Where("booking_id = ?", bookingID).
			Updates(map[string]interface{}{
				"provider":          "paystack",
				"payment_intent_id": reference,
				"status":            models.PaymentPaid,
			}).Error
	})
}

func isSuccessfulTransaction(status string) bool {
	return strings.EqualFold(status, "success")
}

func toMinorUnits(amount float64, currency string) (int64, error) {
	if amount <= 0 {
		return 0, ErrPaymentInvalidAmount
	}
	// Paystack uses kobo for NGN and cents for USD/GHS etc.
	multiplier := 100.0
	if currency == "" {
		currency = "NGN"
	}
	minor := int64(math.Round(amount * multiplier))
	if minor <= 0 {
		return 0, ErrPaymentInvalidAmount
	}
	return minor, nil
}
