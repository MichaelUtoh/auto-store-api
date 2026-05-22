package repositories

import (
	"auto-store-api/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(n *models.Notification) error {
	return r.db.Create(n).Error
}

func (r *NotificationRepository) GetByID(id uuid.UUID) (*models.Notification, error) {
	var n models.Notification
	err := r.db.First(&n, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NotificationRepository) GetByIdempotencyKey(key string) (*models.Notification, error) {
	var n models.Notification
	err := r.db.First(&n, "idempotency_key = ?", key).Error
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NotificationRepository) Update(n *models.Notification) error {
	return r.db.Save(n).Error
}

func (r *NotificationRepository) ListByUser(userID uuid.UUID, inAppOnly bool, unreadOnly bool, page, limit int) ([]models.Notification, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := r.db.Model(&models.Notification{}).Where("user_id = ?", userID)
	if inAppOnly {
		q = q.Where("channel = ?", models.NotificationChannelInApp)
	}
	if unreadOnly {
		q = q.Where("read_at IS NULL")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.Notification
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (r *NotificationRepository) CountUnreadInApp(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.Notification{}).
		Where("user_id = ? AND channel = ? AND read_at IS NULL", userID, models.NotificationChannelInApp).
		Count(&count).Error
	return count, err
}

func (r *NotificationRepository) MarkRead(id, userID uuid.UUID) error {
	now := time.Now()
	res := r.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ? AND channel = ?", id, userID, models.NotificationChannelInApp).
		Update("read_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(userID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.Notification{}).
		Where("user_id = ? AND channel = ? AND read_at IS NULL", userID, models.NotificationChannelInApp).
		Update("read_at", now).Error
}

func (r *NotificationRepository) ListPendingExternal(limit int) ([]models.Notification, error) {
	var items []models.Notification
	err := r.db.Where("channel IN ? AND status IN ?",
		[]models.NotificationChannel{models.NotificationChannelEmail, models.NotificationChannelSMS, models.NotificationChannelPush},
		[]models.NotificationStatus{models.NotificationStatusPending, models.NotificationStatusQueued},
	).Order("created_at ASC").Limit(limit).Find(&items).Error
	return items, err
}

func (r *NotificationRepository) GetOrCreatePreferences(userID uuid.UUID) (*models.NotificationPreference, error) {
	var pref models.NotificationPreference
	err := r.db.First(&pref, "user_id = ?", userID).Error
	if err == nil {
		return &pref, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	pref = models.NotificationPreference{
		UserID:       userID,
		EmailEnabled: true,
		SmsEnabled:   false,
		PushEnabled:  false,
		InAppEnabled: true,
	}
	if err := r.db.Create(&pref).Error; err != nil {
		return nil, err
	}
	return &pref, nil
}

func (r *NotificationRepository) UpdatePreferences(pref *models.NotificationPreference) error {
	return r.db.Save(pref).Error
}
