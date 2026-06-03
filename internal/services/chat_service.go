package services

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"auto-store-api/internal/config"
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"auto-store-api/pkg/auth"
	"auto-store-api/pkg/cache"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrConversationNotFound   = errors.New("conversation not found")
	ErrConversationForbidden  = errors.New("forbidden")
	ErrConversationClosed     = errors.New("conversation is closed")
	ErrGuestSessionRateLimit  = errors.New("too many guest sessions from this address")
	ErrMessageRateLimit       = errors.New("too many messages, please wait")
	ErrInvalidMessageBody     = errors.New("invalid message body")
	ErrInvalidGuestEmail      = errors.New("invalid guest email")
	ErrInvalidGuestToken      = errors.New("invalid guest token")
)

type ChatIdentity struct {
	UserID  *uuid.UUID
	GuestID *uuid.UUID
	User    *models.User
}

func (id ChatIdentity) IsAdmin() bool {
	return id.User != nil && id.User.Role == models.RoleAdmin
}

type CreateConversationInput struct {
	ContextType *models.ChatContextType
	ContextID   *uuid.UUID
	GuestName   string
}

type UpdateConversationInput struct {
	Status     *models.ConversationStatus
	GuestEmail *string
	GuestName  *string
}

type ChatService struct {
	repo      *repositories.ChatRepository
	userRepo  *repositories.UserRepository
	guestJWT  *auth.GuestTokenManager
	cfg       config.ChatConfig
	publisher MessagePublisher
	notifier  *Notifier
	log       *zap.Logger
}

// MessagePublisher broadcasts new messages (WebSocket hub implements this).
type MessagePublisher interface {
	PublishMessage(conversationID uuid.UUID, message *models.ChatMessage)
}

type noopPublisher struct{}

func (noopPublisher) PublishMessage(uuid.UUID, *models.ChatMessage) {}

func NewChatService(
	repo *repositories.ChatRepository,
	userRepo *repositories.UserRepository,
	guestJWT *auth.GuestTokenManager,
	cfg config.ChatConfig,
	publisher MessagePublisher,
	notifier *Notifier,
	log *zap.Logger,
) *ChatService {
	if publisher == nil {
		publisher = noopPublisher{}
	}
	return &ChatService{
		repo:      repo,
		userRepo:  userRepo,
		guestJWT:  guestJWT,
		cfg:       cfg,
		publisher: publisher,
		notifier:  notifier,
		log:       log,
	}
}

func (s *ChatService) CreateGuestSession(ctx context.Context, clientIP string) (uuid.UUID, string, time.Time, error) {
	if err := s.checkGuestSessionRateLimit(ctx, clientIP); err != nil {
		return uuid.Nil, "", time.Time{}, err
	}
	guestID := uuid.New()
	token, expiresAt, err := s.guestJWT.Generate(guestID)
	if err != nil {
		return uuid.Nil, "", time.Time{}, err
	}
	return guestID, token, expiresAt, nil
}

func (s *ChatService) RefreshGuestSession(guestID uuid.UUID) (string, time.Time, error) {
	token, expiresAt, err := s.guestJWT.Generate(guestID)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (s *ChatService) GetOpenConversation(identity ChatIdentity) (*models.Conversation, error) {
	if identity.UserID != nil {
		conv, err := s.repo.GetOpenByUserID(*identity.UserID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return conv, err
	}
	if identity.GuestID != nil {
		conv, err := s.repo.GetOpenByGuestID(*identity.GuestID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return conv, err
	}
	return nil, ErrConversationForbidden
}

func (s *ChatService) GetOrCreateConversation(identity ChatIdentity, input CreateConversationInput) (*models.Conversation, bool, error) {
	if identity.UserID != nil {
		if conv, err := s.repo.GetOpenByUserID(*identity.UserID); err == nil {
			return conv, false, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, err
		}
		conv := &models.Conversation{
			UserID:        identity.UserID,
			Status:        models.ConversationStatusOpen,
			ContextType:   input.ContextType,
			ContextID:     input.ContextID,
			LastMessageAt: time.Now(),
		}
		return conv, true, s.repo.CreateConversation(conv)
	}
	if identity.GuestID != nil {
		if conv, err := s.repo.GetOpenByGuestID(*identity.GuestID); err == nil {
			return conv, false, nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, err
		}
		conv := &models.Conversation{
			GuestID:       identity.GuestID,
			GuestName:     strings.TrimSpace(input.GuestName),
			Status:        models.ConversationStatusOpen,
			ContextType:   input.ContextType,
			ContextID:     input.ContextID,
			LastMessageAt: time.Now(),
		}
		return conv, true, s.repo.CreateConversation(conv)
	}
	return nil, false, ErrConversationForbidden
}

func (s *ChatService) GetConversation(id uuid.UUID, identity ChatIdentity) (*models.Conversation, error) {
	conv, err := s.repo.GetConversationByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, err
	}
	if !s.canAccess(conv, identity) {
		return nil, ErrConversationForbidden
	}
	return conv, nil
}

func (s *ChatService) ListMessages(conversationID uuid.UUID, identity ChatIdentity, page, limit int, since *time.Time) ([]models.ChatMessage, int64, error) {
	if _, err := s.GetConversation(conversationID, identity); err != nil {
		return nil, 0, err
	}
	return s.repo.ListMessages(conversationID, page, limit, since)
}

func (s *ChatService) SendMessage(ctx context.Context, conversationID uuid.UUID, identity ChatIdentity, body string) (*models.ChatMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" || len(body) > s.cfg.MessageMaxLen {
		return nil, ErrInvalidMessageBody
	}
	if err := s.checkMessageRateLimit(ctx, identity); err != nil {
		return nil, err
	}

	conv, err := s.GetConversation(conversationID, identity)
	if err != nil {
		return nil, err
	}
	if conv.Status == models.ConversationStatusClosed {
		return nil, ErrConversationClosed
	}

	msg := &models.ChatMessage{
		ConversationID: conversationID,
		Body:           body,
		CreatedAt:      time.Now(),
	}

	if identity.IsAdmin() {
		msg.SenderType = models.MessageSenderAdmin
		msg.SenderUserID = &identity.User.ID
	} else {
		msg.SenderType = models.MessageSenderCustomer
		if identity.UserID != nil {
			msg.SenderUserID = identity.UserID
		}
	}

	if err := s.repo.CreateMessage(msg); err != nil {
		return nil, err
	}
	_ = s.repo.TouchLastMessageAt(conversationID, msg.CreatedAt)
	s.publisher.PublishMessage(conversationID, msg)
	s.dispatchMessageNotifications(ctx, conv, msg, identity)
	return msg, nil
}

func (s *ChatService) dispatchMessageNotifications(ctx context.Context, conv *models.Conversation, msg *models.ChatMessage, identity ChatIdentity) {
	if s.notifier == nil {
		return
	}
	if identity.IsAdmin() {
		if err := s.notifier.SupportAdminReplied(ctx, conv, msg.ID, msg.Body); err != nil && s.log != nil {
			s.log.Warn("support admin replied notification failed", zap.Error(err), zap.String("conversation_id", conv.ID.String()))
		}
		return
	}
	count, err := s.repo.CountMessages(conv.ID)
	if err != nil || count != 1 || s.userRepo == nil {
		return
	}
	admins, err := s.userRepo.ListByRole(models.RoleAdmin)
	if err != nil {
		if s.log != nil {
			s.log.Warn("list admins for support notification failed", zap.Error(err))
		}
		return
	}
	for _, admin := range admins {
		if err := s.notifier.SupportNewConversation(ctx, admin.ID, conv, msg.Body); err != nil && s.log != nil {
			s.log.Warn("support new conversation notification failed", zap.Error(err), zap.String("admin_id", admin.ID.String()))
		}
	}
}

func (s *ChatService) UpdateConversation(conversationID uuid.UUID, identity ChatIdentity, input UpdateConversationInput) (*models.Conversation, error) {
	conv, err := s.GetConversation(conversationID, identity)
	if err != nil {
		return nil, err
	}

	if input.Status != nil {
		if *input.Status != models.ConversationStatusClosed {
			return nil, ErrInvalidMessageBody
		}
		conv.Status = models.ConversationStatusClosed
	}

	if input.GuestEmail != nil {
		if identity.GuestID == nil {
			return nil, ErrConversationForbidden
		}
		email := strings.TrimSpace(*input.GuestEmail)
		if email == "" {
			return nil, ErrInvalidGuestEmail
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return nil, ErrInvalidGuestEmail
		}
		conv.GuestEmail = email
	}
	if input.GuestName != nil {
		if identity.GuestID == nil && !identity.IsAdmin() {
			return nil, ErrConversationForbidden
		}
		conv.GuestName = strings.TrimSpace(*input.GuestName)
	}

	if err := s.repo.UpdateConversation(conv); err != nil {
		return nil, err
	}
	return conv, nil
}

func (s *ChatService) MarkRead(conversationID uuid.UUID, identity ChatIdentity) error {
	if _, err := s.GetConversation(conversationID, identity); err != nil {
		return err
	}
	now := time.Now()
	if identity.IsAdmin() {
		return s.repo.MarkAdminRead(conversationID, now)
	}
	return s.repo.MarkCustomerRead(conversationID, now)
}

func (s *ChatService) LinkGuest(userID uuid.UUID, guestToken string) ([]uuid.UUID, error) {
	claims, err := s.guestJWT.Validate(guestToken)
	if err != nil {
		return nil, ErrInvalidGuestToken
	}
	return s.repo.LinkGuestConversations(claims.GuestID, userID)
}

func (s *ChatService) AdminList(params repositories.AdminConversationListParams) ([]models.Conversation, int64, error) {
	return s.repo.ListAdmin(params)
}

func (s *ChatService) AdminUnreadCount() (int64, error) {
	return s.repo.CountAdminUnreadConversations()
}

func (s *ChatService) UnreadCountForViewer(conv *models.Conversation, identity ChatIdentity) (int64, error) {
	if identity.IsAdmin() {
		return s.repo.CountUnreadForAdmin(conv)
	}
	return s.repo.CountUnreadForCustomer(conv)
}

func (s *ChatService) canAccess(conv *models.Conversation, identity ChatIdentity) bool {
	if identity.IsAdmin() {
		return true
	}
	if identity.UserID != nil && conv.UserID != nil && *identity.UserID == *conv.UserID {
		return true
	}
	if identity.GuestID != nil && conv.GuestID != nil && *identity.GuestID == *conv.GuestID {
		return true
	}
	return false
}

func (s *ChatService) checkGuestSessionRateLimit(ctx context.Context, clientIP string) error {
	if cache.Client == nil {
		return nil
	}
	key := "chat:guest_session:" + clientIP
	n, err := cache.Client.Incr(ctx, key).Result()
	if err != nil {
		return nil
	}
	if n == 1 {
		_ = cache.Client.Expire(ctx, key, time.Hour).Err()
	}
	if int(n) > s.cfg.GuestSessionPerHour {
		return ErrGuestSessionRateLimit
	}
	return nil
}

func (s *ChatService) checkMessageRateLimit(ctx context.Context, identity ChatIdentity) error {
	if cache.Client == nil {
		return nil
	}
	key := "chat:msg:"
	switch {
	case identity.User != nil:
		key += "user:" + identity.User.ID.String()
	case identity.GuestID != nil:
		key += "guest:" + identity.GuestID.String()
	default:
		return nil
	}
	n, err := cache.Client.Incr(ctx, key).Result()
	if err != nil {
		return nil
	}
	if n == 1 {
		_ = cache.Client.Expire(ctx, key, time.Minute).Err()
	}
	if int(n) > s.cfg.MessageRateLimitPerM {
		return ErrMessageRateLimit
	}
	return nil
}
