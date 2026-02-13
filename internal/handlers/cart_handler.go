package handlers

import (
	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CartHandler struct {
	cart *services.CartService
}

func NewCartHandler(cart *services.CartService) *CartHandler {
	return &CartHandler{cart: cart}
}

// GetCart godoc
// @Summary Get user's cart
// @Tags cart
// @Security BearerAuth
// @Produce json
// @Success 200 {array} object
// @Router /api/v1/cart [get]
func (h *CartHandler) Get(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	items, err := h.cart.GetCart(userID)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, items)
}

// AddCartItem godoc
// @Summary Add item to cart
// @Tags cart
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.AddCartItemRequest true "Cart item"
// @Success 201 {object} object
// @Failure 400 {object} utils.APIResponse
// @Router /api/v1/cart/items [post]
func (h *CartHandler) AddItem(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	item, err := h.cart.AddItem(userID, req.ProductID, req.Quantity)
	if err != nil {
		if err == services.ErrProductNotFound {
			utils.JSONNotFound(c, "product not found")
			return
		}
		if err == services.ErrInsufficientStock {
			utils.JSONBadRequest(c, "insufficient stock")
			return
		}
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusCreated, item)
}

// UpdateCartItem godoc
// @Summary Update cart item quantity
// @Tags cart
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Cart item ID"
// @Param body body dto.UpdateCartItemRequest true "Quantity"
// @Success 200 {object} utils.APIResponse
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/cart/items/{id} [put]
func (h *CartHandler) UpdateItem(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid item id")
		return
	}
	var req dto.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if err := h.cart.UpdateItem(userID, itemID, req.Quantity); err != nil {
		utils.JSONNotFound(c, "cart item not found")
		return
	}
	utils.JSON(c, http.StatusOK, gin.H{"message": "updated"})
}

// RemoveCartItem godoc
// @Summary Remove item from cart
// @Tags cart
// @Security BearerAuth
// @Param id path string true "Cart item ID"
// @Success 204
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/cart/items/{id} [delete]
func (h *CartHandler) RemoveItem(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid item id")
		return
	}
	if err := h.cart.RemoveItem(userID, itemID); err != nil {
		utils.JSONNotFound(c, "cart item not found")
		return
	}
	c.Status(http.StatusNoContent)
}

// ClearCart godoc
// @Summary Clear cart
// @Tags cart
// @Security BearerAuth
// @Success 204
// @Router /api/v1/cart [delete]
func (h *CartHandler) Clear(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	_ = h.cart.Clear(userID)
	c.Status(http.StatusNoContent)
}
