package handlers

import (
	"context"
	"net/http"
	"strings"

	"auto-store-api/internal/chat"
	"auto-store-api/internal/models"
	"auto-store-api/internal/services"
	"auto-store-api/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var chatUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize:  1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ChatWSHandler struct {
	hub      *chat.Hub
	chat     *services.ChatService
	jwt      *auth.JWTManager
	guestJWT *auth.GuestTokenManager
}

func NewChatWSHandler(hub *chat.Hub, chatSvc *services.ChatService, jwt *auth.JWTManager, guestJWT *auth.GuestTokenManager) *ChatWSHandler {
	return &ChatWSHandler{hub: hub, chat: chatSvc, jwt: jwt, guestJWT: guestJWT}
}

func (h *ChatWSHandler) Handle(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		token = wsBearerToken(c)
	}
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	var userID *uuid.UUID
	var guestID *uuid.UUID
	isAdmin := false

	if claims, err := h.jwt.ValidateAccessToken(token); err == nil {
		uid := claims.UserID
		userID = &uid
		isAdmin = claims.Role == string(models.RoleAdmin)
	} else if guestClaims, err := h.guestJWT.Validate(token); err == nil {
		gid := guestClaims.GuestID
		guestID = &gid
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	conn, err := chatUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	checker := h.accessChecker(isAdmin, userID, guestID)
	sender := h.messageSender(isAdmin, userID, guestID)
	h.hub.HandleConnection(conn, isAdmin, userID, guestID, checker, sender)
}

func (h *ChatWSHandler) messageSender(isAdmin bool, userID, guestID *uuid.UUID) chat.MessageHandler {
	return func(conversationID uuid.UUID, body string, admin bool, uid, gid *uuid.UUID) (*models.ChatMessage, error) {
		identity := h.buildIdentity(admin, uid, gid)
		return h.chat.SendMessage(context.Background(), conversationID, identity, body)
	}
}

func (h *ChatWSHandler) buildIdentity(isAdmin bool, userID, guestID *uuid.UUID) services.ChatIdentity {
	identity := services.ChatIdentity{}
	if userID != nil {
		identity.UserID = userID
		if isAdmin {
			identity.User = &models.User{ID: *userID, Role: models.RoleAdmin}
		}
	}
	if guestID != nil {
		identity.GuestID = guestID
	}
	return identity
}

func (h *ChatWSHandler) accessChecker(isAdmin bool, userID, guestID *uuid.UUID) chat.AccessChecker {
	return func(conversationID uuid.UUID, admin bool, uid, gid *uuid.UUID) bool {
		identity := h.buildIdentity(admin, uid, gid)
		_, err := h.chat.GetConversation(conversationID, identity)
		return err == nil
	}
}

func wsBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
