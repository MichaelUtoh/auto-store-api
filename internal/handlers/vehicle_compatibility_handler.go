package handlers

import (
	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/repositories"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VehicleCompatibilityHandler struct {
	compat *services.VehicleCompatibilityService
}

func NewVehicleCompatibilityHandler(compat *services.VehicleCompatibilityService) *VehicleCompatibilityHandler {
	return &VehicleCompatibilityHandler{compat: compat}
}

// ListVehicleCompatibilities godoc
// @Summary List vehicle compatibility catalog (Admin/Vendor)
// @Tags vehicle-compatibilities
// @Security BearerAuth
// @Produce json
// @Param make query string false "Filter by make"
// @Param model query string false "Filter by model"
// @Param market_variant query string false "Filter by market variant"
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} utils.APIResponse
// @Router /api/v1/vehicle-compatibilities [get]
func (h *VehicleCompatibilityHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	filter := repositories.ListVehicleCompatibilityFilter{
		Make:          c.Query("make"),
		Model:         c.Query("model"),
		MarketVariant: c.Query("market_variant"),
		Page:          page,
		Limit:         limit,
	}
	list, total, err := h.compat.List(filter)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSONPaginated(c, list, filter.Page, filter.Limit, total)
}

// CreateVehicleCompatibility godoc
// @Summary Create a vehicle compatibility catalog entry (Admin/Vendor)
// @Tags vehicle-compatibilities
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.CreateVehicleCompatibilityRequest true "Vehicle compatibility"
// @Success 201 {object} object
// @Router /api/v1/vehicle-compatibilities [post]
func (h *VehicleCompatibilityHandler) Create(c *gin.Context) {
	var req dto.CreateVehicleCompatibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	created, err := h.compat.Create(services.CreateVehicleCompatibilityInput{
		Make:          req.Make,
		Model:         req.Model,
		Generation:    req.Generation,
		YearStart:     req.YearStart,
		YearEnd:       req.YearEnd,
		Engine:        req.Engine,
		Trim:          req.Trim,
		MarketVariant: req.MarketVariant,
		Notes:         req.Notes,
	})
	if err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusCreated, created)
}

// GetVehicleCompatibility godoc
// @Summary Get a vehicle compatibility catalog entry (Admin/Vendor)
// @Tags vehicle-compatibilities
// @Security BearerAuth
// @Produce json
// @Param id path string true "Compatibility ID (UUID)"
// @Success 200 {object} object
// @Router /api/v1/vehicle-compatibilities/{id} [get]
func (h *VehicleCompatibilityHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid compatibility id")
		return
	}
	v, err := h.compat.GetByID(id)
	if err != nil {
		utils.JSONNotFound(c, "compatibility not found")
		return
	}
	utils.JSON(c, http.StatusOK, v)
}
