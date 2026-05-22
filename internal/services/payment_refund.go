package services

import (
	"auto-store-api/internal/models"
	"auto-store-api/pkg/paystack"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrPaymentNotRefundable  = errors.New("payment is not refundable")
	ErrRefundAlreadyInitiated = errors.New("refund already initiated for this payment")
)

// RefundInput optional notes and partial amount (major units, e.g. Naira).
type RefundInput struct {
	CustomerNote string
	MerchantNote string
	Amount       *float64
}

// RefundResult is returned after initiating a Paystack refund.
type RefundResult struct {
	PaystackRefundID     int64  `json:"paystack_refund_id"`
	Status               string `json:"status"`
	Amount               int64  `json:"amount"`
	Currency             string `json:"currency"`
	TransactionReference string `json:"transaction_reference"`
	EntityType           string `json:"entity_type"`
	EntityID             uuid.UUID `json:"entity_id"`
}

func (s *PaymentService) RefundOrder(requesterID, orderID uuid.UUID, asAdmin bool, in RefundInput) (*RefundResult, error) {
	if err := s.requireClient(); err != nil {
		return nil, err
	}
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if !asAdmin && order.UserID != requesterID {
		return nil, ErrOrderNotFound
	}
	return s.refundOrder(order, in)
}

func (s *PaymentService) RefundBooking(requesterID, bookingID uuid.UUID, asAdmin bool, in RefundInput) (*RefundResult, error) {
	if err := s.requireClient(); err != nil {
		return nil, err
	}
	var booking *models.InstallationBooking
	var err error
	if asAdmin {
		booking, err = s.installRepo.GetBookingByID(bookingID)
	} else {
		booking, err = s.installRepo.GetBookingByIDForUser(bookingID, requesterID)
	}
	if err != nil {
		return nil, ErrInstallationBookingNotFound
	}
	return s.refundBooking(booking, in)
}

// RefundBookingOnCancel initiates Paystack refund when cancelling a paid booking.
func (s *PaymentService) RefundBookingOnCancel(booking *models.InstallationBooking, reason string) error {
	if booking.PaymentStatus != models.PaymentPaid {
		return nil
	}
	if !s.cfg.Enabled || s.client == nil {
		return nil
	}
	_, err := s.refundBooking(booking, RefundInput{
		CustomerNote: reason,
		MerchantNote: fmt.Sprintf("Booking %s cancelled", booking.ID),
	})
	return err
}

func (s *PaymentService) refundOrder(order *models.Order, in RefundInput) (*RefundResult, error) {
	if err := validateRefundable(order.PaymentStatus, order.PaymentMethod, order.PaymentReference, order.PaystackRefundID); err != nil {
		return nil, err
	}
	if order.Status == models.OrderStatusShipped || order.Status == models.OrderStatusDelivered {
		return nil, errors.New("cannot refund shipped or delivered orders")
	}

	amountMinor, err := s.refundAmountMinor(order.Total, in.Amount)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.CreateRefund(paystack.RefundParams{
		Transaction:  order.PaymentReference,
		Amount:       amountMinor,
		Currency:     s.cfg.Currency,
		CustomerNote: in.CustomerNote,
		MerchantNote: in.MerchantNote,
	})
	if err != nil {
		return nil, err
	}

	if err := s.markOrderRefunded(order.ID, resp); err != nil {
		return nil, err
	}

	return &RefundResult{
		PaystackRefundID:     resp.Data.ID,
		Status:               resp.Data.Status,
		Amount:               resp.Data.Amount,
		Currency:             s.cfg.Currency,
		TransactionReference: order.PaymentReference,
		EntityType:           paymentEntityOrder,
		EntityID:             order.ID,
	}, nil
}

func (s *PaymentService) refundBooking(booking *models.InstallationBooking, in RefundInput) (*RefundResult, error) {
	if booking.Status == models.BookingStatusCompleted {
		return nil, errors.New("cannot refund completed bookings")
	}

	payment, err := s.getBookingPayment(booking.ID)
	if err != nil {
		return nil, err
	}
	if err := validateRefundable(payment.Status, payment.Provider, payment.PaymentIntentID, payment.PaystackRefundID); err != nil {
		return nil, err
	}

	amountMinor, err := s.refundAmountMinor(booking.TotalAmount, in.Amount)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.CreateRefund(paystack.RefundParams{
		Transaction:  payment.PaymentIntentID,
		Amount:       amountMinor,
		Currency:     s.cfg.Currency,
		CustomerNote: in.CustomerNote,
		MerchantNote: in.MerchantNote,
	})
	if err != nil {
		return nil, err
	}

	if err := s.markBookingRefunded(booking.ID, resp); err != nil {
		return nil, err
	}

	return &RefundResult{
		PaystackRefundID:     resp.Data.ID,
		Status:               resp.Data.Status,
		Amount:               resp.Data.Amount,
		Currency:             s.cfg.Currency,
		TransactionReference: payment.PaymentIntentID,
		EntityType:           paymentEntityBooking,
		EntityID:             booking.ID,
	}, nil
}

func (s *PaymentService) getBookingPayment(bookingID uuid.UUID) (*models.BookingPayment, error) {
	var payment models.BookingPayment
	if err := s.db.First(&payment, "booking_id = ?", bookingID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaymentNotRefundable
		}
		return nil, err
	}
	return &payment, nil
}

func validateRefundable(status models.PaymentStatus, provider, reference, existingRefundID string) error {
	if status != models.PaymentPaid {
		return ErrPaymentNotRefundable
	}
	if provider != "paystack" || reference == "" {
		return ErrPaymentNotRefundable
	}
	if existingRefundID != "" {
		return ErrRefundAlreadyInitiated
	}
	return nil
}

// refundAmountMinor returns minor units for a partial refund, or 0 to refund the full transaction.
func (s *PaymentService) refundAmountMinor(total float64, partial *float64) (int64, error) {
	if partial == nil {
		return 0, nil
	}
	if *partial > total {
		return 0, errors.New("refund amount cannot exceed payment total")
	}
	return toMinorUnits(*partial, s.cfg.Currency)
}

func (s *PaymentService) markOrderRefunded(orderID uuid.UUID, resp *paystack.RefundResponse) error {
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return err
	}
	order.PaymentStatus = models.PaymentRefunded
	order.PaystackRefundID = strconv.FormatInt(resp.Data.ID, 10)
	if order.Status != models.OrderStatusCancelled {
		order.Status = models.OrderStatusCancelled
	}
	return s.orderRepo.Update(order)
}

func (s *PaymentService) markBookingRefunded(bookingID uuid.UUID, resp *paystack.RefundResponse) error {
	booking, err := s.installRepo.GetBookingByID(bookingID)
	if err != nil {
		return err
	}
	booking.PaymentStatus = models.PaymentRefunded
	refundID := strconv.FormatInt(resp.Data.ID, 10)

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(booking).Error; err != nil {
			return err
		}
		return tx.Model(&models.BookingPayment{}).
			Where("booking_id = ?", bookingID).
			Updates(map[string]interface{}{
				"status":             models.PaymentRefunded,
				"paystack_refund_id": refundID,
			}).Error
	})
}

func (s *PaymentService) handleRefundWebhook(data paystack.RefundWebhookData) error {
	ref := data.Transaction.Reference
	if ref == "" {
		return nil
	}
	if !isRefundTerminal(data.Status) {
		return nil
	}

	entityType, entityID, err := s.resolveEntityFromReference(ref)
	if err != nil {
		s.log.Warn("refund webhook: unknown transaction reference", zap.String("reference", ref))
		return nil
	}

	refundID := strconv.FormatInt(data.ID, 10)
	switch entityType {
	case paymentEntityOrder:
		order, err := s.orderRepo.GetByID(entityID)
		if err != nil {
			return err
		}
		if order.PaymentStatus == models.PaymentRefunded {
			return nil
		}
		order.PaymentStatus = models.PaymentRefunded
		order.PaystackRefundID = refundID
		return s.orderRepo.Update(order)
	case paymentEntityBooking:
		return s.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.InstallationBooking{}).Where("id = ?", entityID).
				Update("payment_status", models.PaymentRefunded).Error; err != nil {
				return err
			}
			return tx.Model(&models.BookingPayment{}).Where("booking_id = ?", entityID).
				Updates(map[string]interface{}{
					"status":             models.PaymentRefunded,
					"paystack_refund_id": refundID,
				}).Error
		})
	}
	return nil
}

func isRefundTerminal(status string) bool {
	switch status {
	case "processed", "failed":
		return true
	default:
		return false
	}
}
