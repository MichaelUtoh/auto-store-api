package handlers

import (
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type WishlistHandler struct {
	wishlist *services.WishlistService
}

func NewWishlistHandler(wishlist *services.WishlistService) *WishlistHandler {
	return &WishlistHandler{wishlist: wishlist}
}

// GetWishlist godoc
// @Summary Get user's wishlist
// @Tags wishlist
// @Security BearerAuth
// @Produce json
// @Success 200 {array} object
// @Router /api/v1/wishlist [get]
func (h *WishlistHandler) Get(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	items, err := h.wishlist.GetWishlist(userID)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, items)
}

// AddWishlist godoc
// @Summary Add product to wishlist
// @Tags wishlist
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body object{product_id=string} true "Product ID"
// @Success 201 {object} object
// @Failure 400 {object} utils.APIResponse
// @Router /api/v1/wishlist [post]
func (h *WishlistHandler) Add(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req struct {
		ProductID uuid.UUID `json:"product_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	productID := req.ProductID
	item, err := h.wishlist.Add(userID, productID)
	if err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if item == nil {
		utils.JSON(c, http.StatusOK, gin.H{"message": "already in wishlist"})
		return
	}
	utils.JSON(c, http.StatusCreated, item)
}

// RemoveWishlist godoc
// @Summary Remove product from wishlist
// @Tags wishlist
// @Security BearerAuth
// @Param productId path string true "Product ID"
// @Success 204
// @Router /api/v1/wishlist/{productId} [delete]
func (h *WishlistHandler) Remove(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	productID, err := uuid.Parse(c.Param("productId"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	_ = h.wishlist.Remove(userID, productID)
	c.Status(http.StatusNoContent)
}
