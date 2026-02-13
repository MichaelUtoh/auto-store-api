package handlers

import (
	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/models"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"
	"net/http"

	"auto-store-api/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	user *services.UserService
}

func NewUserHandler(user *services.UserService) *UserHandler {
	return &UserHandler{user: user}
}

// GetProfile godoc
// @Summary Get current user profile
// @Tags users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.UserResponse
// @Router /api/v1/users/me [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	user, ok := middleware.GetUser(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	utils.JSON(c, http.StatusOK, dto.UserToResponse(user))
}

// UpdateProfile godoc
// @Summary Update profile
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.UpdateProfileRequest true "Profile data"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} utils.APIResponse
// @Router /api/v1/users/me [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	user, ok := middleware.GetUser(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if err := h.user.UpdateProfile(user, req.FirstName, req.LastName, req.Phone); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	updated, _ := h.user.GetByID(user.ID)
	utils.JSON(c, http.StatusOK, dto.UserToResponse(updated))
}

// ListAddresses godoc
// @Summary Get user addresses
// @Tags users
// @Security BearerAuth
// @Produce json
// @Success 200 {array} object
// @Router /api/v1/users/me/addresses [get]
func (h *UserHandler) ListAddresses(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	addrs, err := h.user.ListAddresses(userID)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, addrs)
}

// AddAddress godoc
// @Summary Add address
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.CreateAddressRequest true "Address"
// @Success 201 {object} object
// @Failure 400 {object} utils.APIResponse
// @Router /api/v1/users/me/addresses [post]
func (h *UserHandler) AddAddress(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.CreateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	addr := &models.Address{
		Type:       models.AddressType(req.Type),
		Street:     req.Street,
		City:       req.City,
		State:      req.State,
		PostalCode: req.PostalCode,
		Country:    req.Country,
		IsDefault:  req.IsDefault,
	}
	if err := h.user.AddAddress(userID, addr); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusCreated, addr)
}

// UpdateAddress godoc
// @Summary Update address
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Address ID"
// @Param body body dto.UpdateAddressRequest true "Address data"
// @Success 200 {object} object
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/users/me/addresses/{id} [put]
func (h *UserHandler) UpdateAddress(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid address id")
		return
	}
	addr, err := h.user.GetAddress(id, userID)
	if err != nil {
		utils.JSONNotFound(c, "address not found")
		return
	}
	var req dto.UpdateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if req.Street != nil {
		addr.Street = *req.Street
	}
	if req.City != nil {
		addr.City = *req.City
	}
	if req.State != nil {
		addr.State = *req.State
	}
	if req.PostalCode != nil {
		addr.PostalCode = *req.PostalCode
	}
	if req.Country != nil {
		addr.Country = *req.Country
	}
	if req.IsDefault != nil {
		addr.IsDefault = *req.IsDefault
	}
	if err := h.user.UpdateAddress(addr); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, addr)
}

// DeleteAddress godoc
// @Summary Delete address
// @Tags users
// @Security BearerAuth
// @Param id path string true "Address ID"
// @Success 204
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/users/me/addresses/{id} [delete]
func (h *UserHandler) DeleteAddress(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid address id")
		return
	}
	if err := h.user.DeleteAddress(id, userID); err != nil {
		utils.JSONNotFound(c, "address not found")
		return
	}
	c.Status(http.StatusNoContent)
}

// UpdateRole godoc
// @Summary Update user role (Admin only)
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param body body dto.UpdateRoleRequest true "Role (ADMIN, VENDOR, CUSTOMER)"
// @Success 200 {object} dto.UserResponse
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/admin/users/{id}/role [put]
func (h *UserHandler) UpdateRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid user id")
		return
	}
	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if err := h.user.UpdateRole(id, models.Role(req.Role)); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	user, err := h.user.GetByID(id)
	if err != nil {
		utils.JSONNotFound(c, "user not found")
		return
	}
	utils.JSON(c, http.StatusOK, dto.UserToResponse(user))
}
