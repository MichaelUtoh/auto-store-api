package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/middleware"
	"auto-store-api/internal/models"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MechanicHandler struct {
	mechanic *services.MechanicService
}

func NewMechanicHandler(mechanic *services.MechanicService) *MechanicHandler {
	return &MechanicHandler{mechanic: mechanic}
}

// Apply godoc
// @Summary Apply to become a verified mechanic
// @Tags mechanics
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.MechanicApplyRequest true "Application"
// @Success 201 {object} dto.MechanicProfileResponse
// @Failure 400,409 {object} utils.APIResponse
// @Router /api/v1/mechanic/apply [post]
func (h *MechanicHandler) Apply(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.MechanicApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	profile, err := h.mechanic.Apply(userID, services.MechanicApplyInput{
		BusinessName:    req.BusinessName,
		Bio:             req.Bio,
		Phone:           req.Phone,
		Street:          req.Street,
		City:            req.City,
		State:           req.State,
		PostalCode:      req.PostalCode,
		Country:         req.Country,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		ServiceRadiusKm: req.ServiceRadiusKm,
		Documents:       toMechanicDocumentInputs(req.Documents),
	})
	if err != nil {
		if errors.Is(err, services.ErrMechanicProfileExists) {
			utils.JSON(c, http.StatusConflict, gin.H{"success": false, "error": err.Error()})
			return
		}
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusCreated, dto.MechanicProfileToResponse(profile))
}

// GetMyProfile godoc
// @Summary Get current user's mechanic profile
// @Tags mechanics
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.MechanicProfileResponse
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/mechanic/profile [get]
func (h *MechanicHandler) GetMyProfile(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	profile, err := h.mechanic.GetProfileByUserID(userID)
	if err != nil {
		if errors.Is(err, services.ErrMechanicProfileNotFound) {
			utils.JSONNotFound(c, err.Error())
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.MechanicProfileToResponse(profile))
}

// UpdateMyProfile godoc
// @Summary Update current user's mechanic profile
// @Tags mechanics
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.MechanicUpdateProfileRequest true "Profile updates"
// @Success 200 {object} dto.MechanicProfileResponse
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/mechanic/profile [put]
func (h *MechanicHandler) UpdateMyProfile(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.MechanicUpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	profile, err := h.mechanic.UpdateOwnProfile(userID, services.MechanicUpdateProfileInput{
		BusinessName:    req.BusinessName,
		Bio:             req.Bio,
		Phone:           req.Phone,
		Street:          req.Street,
		City:            req.City,
		State:           req.State,
		PostalCode:      req.PostalCode,
		Country:         req.Country,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		ServiceRadiusKm: req.ServiceRadiusKm,
	})
	if err != nil {
		if errors.Is(err, services.ErrMechanicProfileNotFound) {
			utils.JSONNotFound(c, err.Error())
			return
		}
		if errors.Is(err, services.ErrMechanicProfileNotEditable) {
			utils.JSONBadRequest(c, err.Error())
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.MechanicProfileToResponse(profile))
}

// AddDocument godoc
// @Summary Add a verification document to mechanic profile
// @Tags mechanics
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.MechanicDocumentRequest true "Document"
// @Success 201 {object} dto.MechanicDocumentResponse
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/mechanic/documents [post]
func (h *MechanicHandler) AddDocument(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.MechanicDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	doc, err := h.mechanic.AddDocument(userID, models.MechanicDocumentType(req.DocumentType), req.URL, req.FileName)
	if err != nil {
		if errors.Is(err, services.ErrMechanicProfileNotFound) {
			utils.JSONNotFound(c, err.Error())
			return
		}
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusCreated, dto.MechanicDocumentToResponse(doc))
}

// RemoveDocument godoc
// @Summary Remove a verification document
// @Tags mechanics
// @Security BearerAuth
// @Param id path string true "Document ID"
// @Success 204
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/mechanic/documents/{id} [delete]
func (h *MechanicHandler) RemoveDocument(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid document id")
		return
	}
	if err := h.mechanic.RemoveDocument(userID, docID); err != nil {
		if errors.Is(err, services.ErrMechanicDocumentNotFound) || errors.Is(err, services.ErrMechanicProfileNotFound) {
			utils.JSONNotFound(c, err.Error())
			return
		}
		utils.JSONBadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// GetPublicProfile godoc
// @Summary Get a verified mechanic's public profile
// @Tags mechanics
// @Produce json
// @Param id path string true "Mechanic profile ID"
// @Success 200 {object} dto.MechanicProfileResponse
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/mechanics/{id} [get]
func (h *MechanicHandler) GetPublicProfile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid mechanic profile id")
		return
	}
	profile, err := h.mechanic.GetPublicProfile(id)
	if err != nil {
		utils.JSONNotFound(c, "mechanic not found")
		return
	}
	utils.JSON(c, http.StatusOK, dto.MechanicProfileToPublicResponse(profile))
}

// ListVerified godoc
// @Summary List verified mechanics
// @Tags mechanics
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {array} dto.MechanicProfileResponse
// @Router /api/v1/mechanics [get]
func (h *MechanicHandler) ListVerified(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	profiles, total, err := h.mechanic.ListVerified(page, limit)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.MechanicProfileResponse, len(profiles))
	for i := range profiles {
		resp[i] = dto.MechanicProfileToPublicResponse(&profiles[i])
	}
	utils.JSONPaginated(c, resp, page, limit, total)
}

// ListAdmin godoc
// @Summary List mechanic profiles (admin)
// @Tags mechanics
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status (pending, verified, suspended, rejected)"
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {array} dto.MechanicProfileResponse
// @Router /api/v1/admin/mechanics [get]
func (h *MechanicHandler) ListAdmin(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	var status *models.MechanicProfileStatus
	if s := c.Query("status"); s != "" {
		st := models.MechanicProfileStatus(s)
		if !models.IsValidMechanicProfileStatus(st) {
			utils.JSONBadRequest(c, "invalid status filter")
			return
		}
		status = &st
	}
	profiles, total, err := h.mechanic.ListForAdmin(status, page, limit)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.MechanicProfileResponse, len(profiles))
	for i := range profiles {
		resp[i] = dto.MechanicProfileToResponse(&profiles[i])
	}
	utils.JSONPaginated(c, resp, page, limit, total)
}

// Verify godoc
// @Summary Verify a mechanic application (admin)
// @Tags mechanics
// @Security BearerAuth
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} dto.MechanicProfileResponse
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/admin/mechanics/{userId}/verify [put]
func (h *MechanicHandler) Verify(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid user id")
		return
	}
	profile, err := h.mechanic.Verify(userID)
	if err != nil {
		if errors.Is(err, services.ErrMechanicProfileNotFound) {
			utils.JSONNotFound(c, err.Error())
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.MechanicProfileToResponse(profile))
}

// Suspend godoc
// @Summary Suspend a mechanic (admin)
// @Tags mechanics
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param body body dto.MechanicAdminActionRequest false "Optional reason"
// @Success 200 {object} dto.MechanicProfileResponse
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/admin/mechanics/{userId}/suspend [put]
func (h *MechanicHandler) Suspend(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid user id")
		return
	}
	var req dto.MechanicAdminActionRequest
	_ = c.ShouldBindJSON(&req)
	profile, err := h.mechanic.Suspend(userID, req.Reason)
	if err != nil {
		if errors.Is(err, services.ErrMechanicProfileNotFound) {
			utils.JSONNotFound(c, err.Error())
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.MechanicProfileToResponse(profile))
}

// Reject godoc
// @Summary Reject a mechanic application (admin)
// @Tags mechanics
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param body body dto.MechanicAdminActionRequest true "Rejection reason"
// @Success 200 {object} dto.MechanicProfileResponse
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/admin/mechanics/{userId}/reject [put]
func (h *MechanicHandler) Reject(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid user id")
		return
	}
	var req dto.MechanicAdminActionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Reason == "" {
		utils.JSONBadRequest(c, "reason is required")
		return
	}
	profile, err := h.mechanic.Reject(userID, req.Reason)
	if err != nil {
		if errors.Is(err, services.ErrMechanicProfileNotFound) {
			utils.JSONNotFound(c, err.Error())
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.MechanicProfileToResponse(profile))
}

func toMechanicDocumentInputs(docs []dto.MechanicDocumentRequest) []services.MechanicDocumentInput {
	if len(docs) == 0 {
		return nil
	}
	out := make([]services.MechanicDocumentInput, len(docs))
	for i, d := range docs {
		out[i] = services.MechanicDocumentInput{
			DocumentType: models.MechanicDocumentType(d.DocumentType),
			URL:          d.URL,
			FileName:     d.FileName,
		}
	}
	return out
}

// RequireVerifiedMechanic is a handler helper used by future mechanic-only routes.
func RequireVerifiedMechanic(c *gin.Context, mechanicSvc *services.MechanicService) (*models.MechanicProfile, bool) {
	user, ok := middleware.GetUser(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return nil, false
	}
	if user.Role != models.RoleMechanic {
		utils.JSONForbidden(c, "mechanic role required")
		return nil, false
	}
	profile, err := mechanicSvc.GetProfileByUserID(user.ID)
	if err != nil || !profile.IsVerified() {
		utils.JSONForbidden(c, "verified mechanic profile required")
		return nil, false
	}
	return profile, true
}
