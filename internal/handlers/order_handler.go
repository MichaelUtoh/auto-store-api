package handlers

import (
	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/models"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrderHandler struct {
	order *services.OrderService
}

func NewOrderHandler(order *services.OrderService) *OrderHandler {
	return &OrderHandler{order: order}
}

// CreateOrder godoc
// @Summary Create order from cart
// @Tags orders
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.CreateOrderRequest true "Order data"
// @Success 201 {object} object
// @Failure 400 {object} utils.APIResponse
// @Router /api/v1/orders [post]
func (h *OrderHandler) Create(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	order, err := h.order.Create(userID, req.ShippingAddressID, req.BillingAddressID, req.PaymentMethod)
	if err != nil {
		if err == services.ErrEmptyCart {
			utils.JSONBadRequest(c, "cart is empty")
			return
		}
		if err == services.ErrAddressNotFound {
			utils.JSONBadRequest(c, "invalid shipping or billing address")
			return
		}
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusCreated, order)
}

// ListOrders godoc
// @Summary List user's orders
// @Tags orders
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} utils.APIResponse
// @Router /api/v1/orders [get]
func (h *OrderHandler) List(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	orders, total, err := h.order.ListByUser(userID, page, limit)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSONPaginated(c, orders, page, limit, total)
}

// GetOrder godoc
// @Summary Get order details
// @Tags orders
// @Security BearerAuth
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} object
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/orders/{id} [get]
func (h *OrderHandler) Get(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid order id")
		return
	}
	order, err := h.order.GetByID(id, userID)
	if err != nil {
		utils.JSONNotFound(c, "order not found")
		return
	}
	utils.JSON(c, http.StatusOK, order)
}

// CancelOrder godoc
// @Summary Cancel order
// @Tags orders
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/orders/{id}/cancel [put]
func (h *OrderHandler) Cancel(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid order id")
		return
	}
	if err := h.order.Cancel(id, userID); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, gin.H{"message": "order cancelled"})
}

// ListAllOrders godoc
// @Summary List all orders (Admin)
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Param status query string false "Order status"
// @Success 200 {object} utils.APIResponse
// @Router /api/v1/admin/orders [get]
func (h *OrderHandler) ListAll(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")
	orders, total, err := h.order.ListAll(page, limit, status)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSONPaginated(c, orders, page, limit, total)
}

// UpdateOrderStatus godoc
// @Summary Update order status (Admin)
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param body body dto.UpdateOrderStatusRequest true "Status"
// @Success 200 {object} object
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/admin/orders/{id}/status [put]
func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid order id")
		return
	}
	var req dto.UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	status := models.OrderStatus(req.Status)
	order, err := h.order.UpdateStatus(id, status)
	if err != nil {
		utils.JSONNotFound(c, "order not found")
		return
	}
	utils.JSON(c, http.StatusOK, order)
}
