package handlers

import (
	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PaymentHandler struct {
	payments *services.PaymentService
	payouts  *services.MechanicPayoutService
}

func NewPaymentHandler(payments *services.PaymentService, payouts *services.MechanicPayoutService) *PaymentHandler {
	return &PaymentHandler{payments: payments, payouts: payouts}
}

// InitializeOrderPayment godoc
// @Summary Initialize Paystack checkout for an order
// @Tags payments
// @Security BearerAuth
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} services.PaymentInitializeResult
// @Router /api/v1/orders/{id}/pay [post]
func (h *PaymentHandler) InitializeOrderPayment(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid order id")
		return
	}
	result, err := h.payments.InitializeOrderPayment(userID, orderID)
	if err != nil {
		h.handlePaymentError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, result)
}

// InitializeBookingPayment godoc
// @Summary Initialize Paystack checkout for an installation booking
// @Tags payments
// @Security BearerAuth
// @Produce json
// @Param id path string true "Booking ID"
// @Success 200 {object} services.PaymentInitializeResult
// @Router /api/v1/installation/bookings/{id}/pay [post]
func (h *PaymentHandler) InitializeBookingPayment(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid booking id")
		return
	}
	result, err := h.payments.InitializeBookingPayment(userID, bookingID)
	if err != nil {
		h.handlePaymentError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, result)
}

// VerifyPayment godoc
// @Summary Verify a Paystack transaction by reference
// @Tags payments
// @Security BearerAuth
// @Produce json
// @Param reference query string true "Paystack reference"
// @Success 200 {object} object
// @Router /api/v1/payments/verify [get]
func (h *PaymentHandler) VerifyPayment(c *gin.Context) {
	reference := c.Query("reference")
	if reference == "" {
		utils.JSONBadRequest(c, "reference is required")
		return
	}
	entityType, entityID, err := h.payments.VerifyPayment(reference)
	if err != nil {
		h.handlePaymentError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, gin.H{
		"entity_type": entityType,
		"entity_id":   entityID,
		"reference":   reference,
		"status":      "paid",
	})
}

// PaystackWebhook godoc
// @Summary Paystack webhook (charge.success)
// @Tags payments
// @Accept json
// @Success 200 {object} object
// @Router /webhooks/paystack [post]
func (h *PaymentHandler) PaystackWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.JSONBadRequest(c, "invalid body")
		return
	}
	signature := c.GetHeader("x-paystack-signature")
	if err := h.payments.HandleWebhook(body, signature); err != nil {
		if err == services.ErrPaystackNotConfigured {
			utils.JSON(c, http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		utils.JSONBadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"received": true})
}

// ListBanks godoc
// @Summary List Paystack-supported banks for payout setup
// @Tags payments
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object
// @Router /api/v1/payments/banks [get]
// RefundOrder godoc
// @Summary Refund a paid order via Paystack
// @Tags payments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param body body dto.RefundRequest false "Optional partial amount and notes"
// @Success 200 {object} services.RefundResult
// @Router /api/v1/orders/{id}/refund [post]
func (h *PaymentHandler) RefundOrder(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid order id")
		return
	}
	in := bindRefundInput(c)
	result, err := h.payments.RefundOrder(userID, orderID, false, in)
	if err != nil {
		h.handlePaymentError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, result)
}

// RefundBooking godoc
// @Summary Refund a paid installation booking via Paystack
// @Tags payments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Booking ID"
// @Param body body dto.RefundRequest false "Optional partial amount and notes"
// @Success 200 {object} services.RefundResult
// @Router /api/v1/installation/bookings/{id}/refund [post]
func (h *PaymentHandler) RefundBooking(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid booking id")
		return
	}
	in := bindRefundInput(c)
	result, err := h.payments.RefundBooking(userID, bookingID, false, in)
	if err != nil {
		h.handlePaymentError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, result)
}

// AdminRefundOrder godoc
// @Summary Admin: refund order via Paystack
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} services.RefundResult
// @Router /api/v1/admin/orders/{id}/refund [post]
func (h *PaymentHandler) AdminRefundOrder(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid order id")
		return
	}
	in := bindRefundInput(c)
	result, err := h.payments.RefundOrder(uuid.Nil, orderID, true, in)
	if err != nil {
		h.handlePaymentError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, result)
}

// AdminRefundBooking godoc
// @Summary Admin: refund installation booking via Paystack
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Booking ID"
// @Success 200 {object} services.RefundResult
// @Router /api/v1/admin/installation/bookings/{id}/refund [post]
func (h *PaymentHandler) AdminRefundBooking(c *gin.Context) {
	bookingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid booking id")
		return
	}
	in := bindRefundInput(c)
	result, err := h.payments.RefundBooking(uuid.Nil, bookingID, true, in)
	if err != nil {
		h.handlePaymentError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, result)
}

func bindRefundInput(c *gin.Context) services.RefundInput {
	var req dto.RefundRequest
	_ = c.ShouldBindJSON(&req)
	return services.RefundInput{
		CustomerNote: req.CustomerNote,
		MerchantNote: req.MerchantNote,
		Amount:       req.Amount,
	}
}

func (h *PaymentHandler) ListBanks(c *gin.Context) {
	banks, err := h.payouts.ListBanks()
	if err != nil {
		h.handlePaymentError(c, err)
		return
	}
	utils.JSON(c, http.StatusOK, banks)
}

func (h *PaymentHandler) handlePaymentError(c *gin.Context, err error) {
	switch err {
	case services.ErrPaystackNotConfigured:
		utils.JSON(c, http.StatusServiceUnavailable, gin.H{"error": err.Error()})
	case services.ErrOrderNotFound, services.ErrInstallationBookingNotFound:
		utils.JSONNotFound(c, err.Error())
	case services.ErrPaymentNotPending, services.ErrBookingSplitInvalid,
		services.ErrPaymentNotRefundable, services.ErrRefundAlreadyInitiated:
		utils.JSONBadRequest(c, err.Error())
	case services.ErrMechanicPayoutNotConfigured:
		utils.JSON(c, http.StatusConflict, gin.H{"error": err.Error()})
	default:
		utils.JSONBadRequest(c, err.Error())
	}
}
