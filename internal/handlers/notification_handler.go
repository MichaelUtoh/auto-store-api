package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NotificationHandler struct {
	notifications *services.NotificationService
}

func NewNotificationHandler(notifications *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifications: notifications}
}

// List godoc
// @Summary List in-app notifications
// @Tags notifications
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Param unread_only query bool false "Unread only"
// @Success 200 {array} dto.NotificationResponse
// @Router /api/v1/notifications [get]
func (h *NotificationHandler) List(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	unreadOnly := c.Query("unread_only") == "true" || c.Query("unread_only") == "1"

	items, total, err := h.notifications.ListInApp(userID, unreadOnly, page, limit)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.NotificationResponse, len(items))
	for i := range items {
		resp[i] = dto.NotificationToResponse(&items[i])
	}
	utils.JSONPaginated(c, resp, page, limit, total)
}

// UnreadCount godoc
// @Summary Get unread in-app notification count
// @Tags notifications
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.UnreadCountResponse
// @Router /api/v1/notifications/unread-count [get]
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	count, err := h.notifications.UnreadCount(userID)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.UnreadCountResponse{Count: count})
}

// MarkRead godoc
// @Summary Mark one in-app notification as read
// @Tags notifications
// @Security BearerAuth
// @Param id path string true "Notification ID"
// @Success 204
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/notifications/{id}/read [patch]
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid notification id")
		return
	}
	if err := h.notifications.MarkRead(id, userID); err != nil {
		if errors.Is(err, services.ErrNotificationNotFound) {
			utils.JSONNotFound(c, err.Error())
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// MarkAllRead godoc
// @Summary Mark all in-app notifications as read
// @Tags notifications
// @Security BearerAuth
// @Success 204
// @Router /api/v1/notifications/read-all [patch]
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	if err := h.notifications.MarkAllRead(userID); err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// GetPreferences godoc
// @Summary Get notification preferences
// @Tags notifications
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.NotificationPreferenceResponse
// @Router /api/v1/users/me/notification-preferences [get]
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	pref, err := h.notifications.GetPreferences(userID)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.NotificationPreferenceToResponse(pref))
}

// UpdatePreferences godoc
// @Summary Update notification preferences
// @Tags notifications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.UpdateNotificationPreferenceRequest true "Preferences"
// @Success 200 {object} dto.NotificationPreferenceResponse
// @Router /api/v1/users/me/notification-preferences [put]
func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.UpdateNotificationPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	pref, err := h.notifications.UpdatePreferences(userID, req.EmailEnabled, req.SmsEnabled, req.PushEnabled, req.InAppEnabled)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.NotificationPreferenceToResponse(pref))
}
