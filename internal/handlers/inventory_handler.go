package handlers

import (
	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/middleware"
	"auto-store-api/internal/models"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type InventoryHandler struct {
	inventory *services.InventoryService
}

func NewInventoryHandler(inventory *services.InventoryService) *InventoryHandler {
	return &InventoryHandler{inventory: inventory}
}

// ListLowStock godoc
// @Summary List low-stock products
// @Tags inventory
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} utils.APIResponse
// @Router /api/v1/admin/inventory/low-stock [get]
func (h *InventoryHandler) ListLowStock(c *gin.Context) {
	user, ok := middleware.GetUser(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	products, total, err := h.inventory.ListLowStock(user, page, limit)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.ProductResponse, len(products))
	for i := range products {
		resp[i] = dto.ProductToResponse(&products[i])
	}
	utils.JSONPaginated(c, resp, page, limit, total)
}

// ListMovements godoc
// @Summary List stock movements for a product
// @Tags inventory
// @Security BearerAuth
// @Produce json
// @Param id path string true "Product ID"
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} utils.APIResponse
// @Router /api/v1/admin/inventory/products/{id}/movements [get]
func (h *InventoryHandler) ListMovements(c *gin.Context) {
	user, ok := middleware.GetUser(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	movements, total, err := h.inventory.ListMovements(user, productID, page, limit)
	if err != nil {
		if err == services.ErrInventoryProductNotFound {
			utils.JSONNotFound(c, "product not found")
			return
		}
		if err == services.ErrInventoryAccessDenied {
			utils.JSONForbidden(c, "access denied")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSONPaginated(c, movements, page, limit, total)
}

// AdjustStock godoc
// @Summary Adjust product stock (restock or correction)
// @Tags inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param body body dto.AdjustStockRequest true "Stock adjustment"
// @Success 200 {object} dto.ProductResponse
// @Router /api/v1/admin/inventory/products/{id}/stock [patch]
func (h *InventoryHandler) AdjustStock(c *gin.Context) {
	user, ok := middleware.GetUser(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	if _, err := h.inventory.GetProductForUser(user, productID); err != nil {
		if err == services.ErrInventoryProductNotFound {
			utils.JSONNotFound(c, "product not found")
			return
		}
		if err == services.ErrInventoryAccessDenied {
			utils.JSONForbidden(c, "access denied")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	var req dto.AdjustStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	reason := models.StockMovementRestock
	if req.Reason != "" {
		reason = models.StockMovementReason(req.Reason)
	}
	performedBy := user.ID
	product, err := h.inventory.AdjustStock(c.Request.Context(), nil, services.AdjustStockInput{
		ProductID:   productID,
		Delta:       req.Delta,
		Reason:      reason,
		PerformedBy: &performedBy,
		Notes:       req.Notes,
	})
	if err != nil {
		if err == services.ErrInvalidStockDelta {
			utils.JSONBadRequest(c, "stock quantity cannot be negative")
			return
		}
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.ProductToResponse(product))
}

// UpdateInventorySettings godoc
// @Summary Update inventory settings for a product
// @Tags inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param body body dto.UpdateInventorySettingsRequest true "Settings"
// @Success 200 {object} dto.ProductResponse
// @Router /api/v1/admin/inventory/products/{id}/settings [put]
func (h *InventoryHandler) UpdateSettings(c *gin.Context) {
	user, ok := middleware.GetUser(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	var req dto.UpdateInventorySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	product, err := h.inventory.UpdateSettings(user, productID, req.LowStockThreshold)
	if err != nil {
		if err == services.ErrInventoryProductNotFound {
			utils.JSONNotFound(c, "product not found")
			return
		}
		if err == services.ErrInventoryAccessDenied {
			utils.JSONForbidden(c, "access denied")
			return
		}
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.ProductToResponse(product))
}

// BulkSetThreshold godoc
// @Summary Bulk-set low stock threshold (Admin)
// @Tags inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.BulkThresholdRequest true "Bulk threshold"
// @Success 200 {object} object
// @Router /api/v1/admin/inventory/bulk-threshold [post]
func (h *InventoryHandler) BulkSetThreshold(c *gin.Context) {
	var req dto.BulkThresholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	count, err := h.inventory.BulkSetThreshold(services.BulkThresholdInput{
		Threshold:  req.Threshold,
		CategoryID: req.CategoryID,
		ProductIDs: req.ProductIDs,
	})
	if err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, gin.H{"updated": count})
}
