package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/middleware"
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"
	"auto-store-api/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatHandler struct {
	chat     *services.ChatService
	jwt      *auth.JWTManager
	guestJWT *auth.GuestTokenManager
}

func NewChatHandler(chat *services.ChatService, jwt *auth.JWTManager, guestJWT *auth.GuestTokenManager) *ChatHandler {
	return &ChatHandler{chat: chat, jwt: jwt, guestJWT: guestJWT}
}

// CreateGuestSession godoc
// @Summary Create guest chat session
// @Tags support-chat
// @Produce json
// @Success 201 {object} dto.GuestSessionResponse
// @Failure 429 {object} utils.APIResponse
// @Router /api/v1/chat/guest-session [post]
func (h *ChatHandler) CreateGuestSession(c *gin.Context) {
	guestID, token, expiresAt, err := h.chat.CreateGuestSession(c.Request.Context(), c.ClientIP())
	if err != nil {
		if errors.Is(err, services.ErrGuestSessionRateLimit) {
			utils.JSON(c, http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusCreated, dto.GuestSessionResponse{
		GuestID:    guestID,
		GuestToken: token,
		ExpiresAt:  expiresAt,
	})
}

// RefreshGuestSession godoc
// @Summary Refresh guest chat token
// @Tags support-chat
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.GuestSessionResponse
// @Router /api/v1/chat/guest-session/refresh [post]
func (h *ChatHandler) RefreshGuestSession(c *gin.Context) {
	guestID, ok := middleware.GetGuestID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	token, expiresAt, err := h.chat.RefreshGuestSession(guestID)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.GuestSessionResponse{
		GuestID:    guestID,
		GuestToken: token,
		ExpiresAt:  expiresAt,
	})
}

// GetMyConversation godoc
// @Summary Get current open conversation
// @Tags support-chat
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.ConversationResponse
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/conversations/me [get]
func (h *ChatHandler) GetMyConversation(c *gin.Context) {
	identity, ok := chatIdentity(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	conv, err := h.chat.GetOpenConversation(identity)
	if err != nil {
		if errors.Is(err, services.ErrConversationNotFound) {
			utils.JSONNotFound(c, "no open conversation")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	unread, _ := h.chat.UnreadCountForViewer(conv, identity)
	utils.JSON(c, http.StatusOK, dto.ConversationToResponse(conv, unread))
}

// CreateOrGetConversation godoc
// @Summary Get or create open support conversation
// @Tags support-chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.CreateConversationRequest false "Optional context"
// @Success 200 {object} dto.ConversationResponse
// @Success 201 {object} dto.ConversationResponse
// @Router /api/v1/conversations [post]
func (h *ChatHandler) CreateOrGetConversation(c *gin.Context) {
	identity, ok := chatIdentity(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.CreateConversationRequest
	_ = c.ShouldBindJSON(&req)
	if req.ContextType != nil && !models.IsValidChatContextType(*req.ContextType) {
		utils.JSONBadRequest(c, "invalid context_type")
		return
	}

	input := services.CreateConversationInput{
		ContextType: req.ContextType,
		ContextID:   req.ContextID,
		GuestName:   req.GuestName,
	}
	conv, created, err := h.chat.GetOrCreateConversation(identity, input)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	unread, _ := h.chat.UnreadCountForViewer(conv, identity)
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	utils.JSON(c, status, dto.ConversationToResponse(conv, unread))
}

func (h *ChatHandler) GetConversation(c *gin.Context) {
	identity, ok := chatIdentity(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid conversation id")
		return
	}
	conv, err := h.chat.GetConversation(id, identity)
	if err != nil {
		h.handleChatError(c, err)
		return
	}
	unread, _ := h.chat.UnreadCountForViewer(conv, identity)
	utils.JSON(c, http.StatusOK, dto.ConversationToResponse(conv, unread))
}

func (h *ChatHandler) ListMessages(c *gin.Context) {
	identity, ok := chatIdentity(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid conversation id")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	var since *time.Time
	if s := c.Query("since"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			utils.JSONBadRequest(c, "invalid since timestamp")
			return
		}
		since = &t
	}

	messages, total, err := h.chat.ListMessages(id, identity, page, limit, since)
	if err != nil {
		h.handleChatError(c, err)
		return
	}
	resp := make([]dto.ChatMessageResponse, len(messages))
	for i := range messages {
		resp[i] = dto.ChatMessageToResponse(&messages[i])
	}
	utils.JSONPaginated(c, resp, page, limit, total)
}

// SendMessage godoc
// @Summary Send a chat message
// @Tags support-chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Param body body dto.SendChatMessageRequest true "Message"
// @Success 201 {object} dto.ChatMessageResponse
// @Router /api/v1/conversations/{id}/messages [post]
func (h *ChatHandler) SendMessage(c *gin.Context) {
	identity, ok := chatIdentity(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid conversation id")
		return
	}
	var req dto.SendChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	msg, err := h.chat.SendMessage(c.Request.Context(), id, identity, req.Body)
	if err != nil {
		h.handleChatError(c, err)
		return
	}
	utils.JSON(c, http.StatusCreated, dto.ChatMessageToResponse(msg))
}

func (h *ChatHandler) UpdateConversation(c *gin.Context) {
	identity, ok := chatIdentity(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid conversation id")
		return
	}
	var req dto.UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if req.Status != nil && *req.Status != models.ConversationStatusClosed {
		utils.JSONBadRequest(c, "invalid status")
		return
	}
	input := services.UpdateConversationInput{
		Status:     req.Status,
		GuestEmail: req.GuestEmail,
		GuestName:  req.GuestName,
	}
	conv, err := h.chat.UpdateConversation(id, identity, input)
	if err != nil {
		h.handleChatError(c, err)
		return
	}
	unread, _ := h.chat.UnreadCountForViewer(conv, identity)
	utils.JSON(c, http.StatusOK, dto.ConversationToResponse(conv, unread))
}

func (h *ChatHandler) MarkRead(c *gin.Context) {
	identity, ok := chatIdentity(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid conversation id")
		return
	}
	if err := h.chat.MarkRead(id, identity); err != nil {
		h.handleChatError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// LinkGuest godoc
// @Summary Link guest chat history to logged-in user
// @Tags support-chat
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.LinkGuestRequest true "Guest token"
// @Success 200 {object} dto.LinkGuestResponse
// @Router /api/v1/conversations/link-guest [post]
func (h *ChatHandler) LinkGuest(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.LinkGuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	ids, err := h.chat.LinkGuest(userID, req.GuestToken)
	if err != nil {
		if errors.Is(err, services.ErrInvalidGuestToken) {
			utils.JSONBadRequest(c, err.Error())
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.LinkGuestResponse{
		LinkedCount:     len(ids),
		ConversationIDs: ids,
	})
}

// AdminList godoc
// @Summary Admin support inbox
// @Tags support-chat
// @Security BearerAuth
// @Produce json
// @Param status query string false "open or closed"
// @Param guest_only query bool false "Guests only"
// @Param unread_only query bool false "Unread only"
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {array} dto.ConversationResponse
// @Router /api/v1/admin/conversations [get]
func (h *ChatHandler) AdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	params := repositories.AdminConversationListParams{
		Page:       page,
		Limit:      limit,
		GuestOnly:  c.Query("guest_only") == "true" || c.Query("guest_only") == "1",
		UnreadOnly: c.Query("unread_only") == "true" || c.Query("unread_only") == "1",
	}
	if st := c.Query("status"); st != "" {
		status := models.ConversationStatus(st)
		if !models.IsValidConversationStatus(status) {
			utils.JSONBadRequest(c, "invalid status")
			return
		}
		params.Status = &status
	}

	conversations, total, err := h.chat.AdminList(params)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	adminIdentity := adminChatIdentity(c)
	resp := make([]dto.ConversationResponse, len(conversations))
	for i := range conversations {
		unread, _ := h.chat.UnreadCountForViewer(&conversations[i], adminIdentity)
		resp[i] = dto.ConversationToResponse(&conversations[i], unread)
	}
	utils.JSONPaginated(c, resp, page, limit, total)
}

func (h *ChatHandler) AdminUnreadCount(c *gin.Context) {
	count, err := h.chat.AdminUnreadCount()
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, gin.H{"count": count})
}

func (h *ChatHandler) handleChatError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrConversationNotFound):
		utils.JSONNotFound(c, err.Error())
	case errors.Is(err, services.ErrConversationForbidden):
		utils.JSON(c, http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrConversationClosed):
		utils.JSONBadRequest(c, err.Error())
	case errors.Is(err, services.ErrMessageRateLimit):
		utils.JSON(c, http.StatusTooManyRequests, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrInvalidMessageBody), errors.Is(err, services.ErrInvalidGuestEmail):
		utils.JSONBadRequest(c, err.Error())
	default:
		utils.JSONInternal(c, err.Error())
	}
}

func chatIdentity(c *gin.Context) (services.ChatIdentity, bool) {
	if user, ok := middleware.GetUser(c); ok {
		uid := user.ID
		return services.ChatIdentity{UserID: &uid, User: user}, true
	}
	if guestID, ok := middleware.GetGuestID(c); ok {
		gid := guestID
		return services.ChatIdentity{GuestID: &gid}, true
	}
	return services.ChatIdentity{}, false
}

func adminChatIdentity(c *gin.Context) services.ChatIdentity {
	user, _ := middleware.GetUser(c)
	if user == nil {
		return services.ChatIdentity{}
	}
	uid := user.ID
	return services.ChatIdentity{UserID: &uid, User: user}
}
