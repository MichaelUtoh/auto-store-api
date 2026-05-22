package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"auto-store-api/internal/config"
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"auto-store-api/pkg/email"
	"auto-store-api/pkg/queue"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
)

// NotifyInput describes a notification to deliver across enabled channels.
type NotifyInput struct {
	UserID         uuid.UUID
	Type           models.NotificationType
	IdempotencyKey string // base key; channel suffix added internally
	Title          string
	Body           string
	Payload        map[string]interface{}
	Channels       []models.NotificationChannel // empty = in_app + email per preferences
}

type NotificationService struct {
	repo        *repositories.NotificationRepository
	userRepo    *repositories.UserRepository
	redis       *redis.Client
	emailSender email.Sender
	cfg         *config.Config
	log         *zap.Logger
}

func NewNotificationService(
	repo *repositories.NotificationRepository,
	userRepo *repositories.UserRepository,
	redisClient *redis.Client,
	emailSender email.Sender,
	cfg *config.Config,
	log *zap.Logger,
) *NotificationService {
	return &NotificationService{
		repo:        repo,
		userRepo:    userRepo,
		redis:       redisClient,
		emailSender: emailSender,
		cfg:         cfg,
		log:         log,
	}
}

// Notify creates delivery records and queues external channel sends.
func (s *NotificationService) Notify(ctx context.Context, input NotifyInput) error {
	if input.UserID == uuid.Nil || input.IdempotencyKey == "" {
		return errors.New("user_id and idempotency_key are required")
	}

	pref, err := s.repo.GetOrCreatePreferences(input.UserID)
	if err != nil {
		return err
	}

	channels := input.Channels
	if len(channels) == 0 {
		channels = defaultChannels(pref)
	}

	payload := input.Payload
	if payload == nil {
		payload = map[string]interface{}{}
	}
	payloadJSON, _ := json.Marshal(payload)

	for _, ch := range channels {
		if !channelAllowed(pref, ch) {
			continue
		}
		key := fmt.Sprintf("%s:%s", input.IdempotencyKey, ch)
		existing, err := s.repo.GetByIdempotencyKey(key)
		if err == nil && existing != nil {
			continue
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		n := &models.Notification{
			UserID:         input.UserID,
			Type:           input.Type,
			Channel:        ch,
			Title:          input.Title,
			Body:           input.Body,
			Payload:        string(payloadJSON),
			IdempotencyKey: key,
		}

		switch ch {
		case models.NotificationChannelInApp:
			now := time.Now()
			n.Status = models.NotificationStatusSent
			n.SentAt = &now
		case models.NotificationChannelEmail, models.NotificationChannelSMS, models.NotificationChannelPush:
			n.Status = models.NotificationStatusQueued
		default:
			continue
		}

		if err := s.repo.Create(n); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				continue
			}
			return err
		}

		if ch != models.NotificationChannelInApp {
			if err := queue.EnqueueNotification(ctx, s.redis, n.ID); err != nil {
				if errors.Is(err, redis.Nil) {
					s.log.Warn("redis unavailable; notification left queued in database", zap.String("id", n.ID.String()))
					n.Status = models.NotificationStatusPending
					_ = s.repo.Update(n)
				} else {
					s.log.Warn("failed to enqueue notification", zap.Error(err), zap.String("id", n.ID.String()))
				}
			}
		}
	}
	return nil
}

func defaultChannels(pref *models.NotificationPreference) []models.NotificationChannel {
	ch := []models.NotificationChannel{}
	if pref.InAppEnabled {
		ch = append(ch, models.NotificationChannelInApp)
	}
	if pref.EmailEnabled {
		ch = append(ch, models.NotificationChannelEmail)
	}
	return ch
}

func channelAllowed(pref *models.NotificationPreference, ch models.NotificationChannel) bool {
	switch ch {
	case models.NotificationChannelInApp:
		return pref.InAppEnabled
	case models.NotificationChannelEmail:
		return pref.EmailEnabled
	case models.NotificationChannelSMS:
		return pref.SmsEnabled
	case models.NotificationChannelPush:
		return pref.PushEnabled
	default:
		return false
	}
}

// ProcessDelivery sends a queued external notification (called by worker).
func (s *NotificationService) ProcessDelivery(ctx context.Context, notificationID uuid.UUID) error {
	n, err := s.repo.GetByID(notificationID)
	if err != nil {
		return err
	}
	if n.Status == models.NotificationStatusSent {
		return nil
	}

	switch n.Channel {
	case models.NotificationChannelEmail:
		user, err := s.userRepo.GetByID(n.UserID)
		if err != nil {
			return err
		}
		subject := n.Title
		body := n.Body
		if s.cfg.App.FrontendURL != "" {
			body += fmt.Sprintf("\n\nView in app: %s/notifications", s.cfg.App.FrontendURL)
		}
		if err := s.emailSender.Send(user.Email, subject, body); err != nil {
			return s.markFailed(n, err)
		}
	case models.NotificationChannelSMS, models.NotificationChannelPush:
		s.log.Info("channel not implemented yet; marking sent", zap.String("channel", string(n.Channel)))
	default:
		return nil
	}

	now := time.Now()
	n.Status = models.NotificationStatusSent
	n.SentAt = &now
	n.FailedReason = ""
	return s.repo.Update(n)
}

func (s *NotificationService) markFailed(n *models.Notification, err error) error {
	n.RetryCount++
	n.FailedReason = err.Error()
	max := s.cfg.Notifications.MaxRetries
	if n.RetryCount >= max {
		n.Status = models.NotificationStatusFailed
		return s.repo.Update(n)
	}
	n.Status = models.NotificationStatusPending
	if updateErr := s.repo.Update(n); updateErr != nil {
		return updateErr
	}
	return err
}

func (s *NotificationService) RequeuePending(ctx context.Context, notificationID uuid.UUID) error {
	return queue.EnqueueNotification(ctx, s.redis, notificationID)
}

func (s *NotificationService) ListInApp(userID uuid.UUID, unreadOnly bool, page, limit int) ([]models.Notification, int64, error) {
	return s.repo.ListByUser(userID, true, unreadOnly, page, limit)
}

func (s *NotificationService) UnreadCount(userID uuid.UUID) (int64, error) {
	return s.repo.CountUnreadInApp(userID)
}

func (s *NotificationService) MarkRead(id, userID uuid.UUID) error {
	if err := s.repo.MarkRead(id, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotificationNotFound
		}
		return err
	}
	return nil
}

func (s *NotificationService) MarkAllRead(userID uuid.UUID) error {
	return s.repo.MarkAllRead(userID)
}

func (s *NotificationService) GetPreferences(userID uuid.UUID) (*models.NotificationPreference, error) {
	return s.repo.GetOrCreatePreferences(userID)
}

func (s *NotificationService) UpdatePreferences(userID uuid.UUID, email, sms, push, inApp *bool) (*models.NotificationPreference, error) {
	pref, err := s.repo.GetOrCreatePreferences(userID)
	if err != nil {
		return nil, err
	}
	if email != nil {
		pref.EmailEnabled = *email
	}
	if sms != nil {
		pref.SmsEnabled = *sms
	}
	if push != nil {
		pref.PushEnabled = *push
	}
	if inApp != nil {
		pref.InAppEnabled = *inApp
	}
	if err := s.repo.UpdatePreferences(pref); err != nil {
		return nil, err
	}
	return pref, nil
}

// PollAndProcessPending re-processes DB-pending notifications when Redis was down.
func (s *NotificationService) PollAndProcessPending(ctx context.Context, limit int) (int, error) {
	items, err := s.repo.ListPendingExternal(limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, item := range items {
		if err := s.ProcessDelivery(ctx, item.ID); err != nil {
			s.log.Warn("pending notification delivery failed", zap.String("id", item.ID.String()), zap.Error(err))
			_ = s.RequeuePending(ctx, item.ID)
			continue
		}
		processed++
	}
	return processed, nil
}
