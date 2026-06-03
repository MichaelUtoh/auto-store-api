package repositories

import (
	"auto-store-api/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) CreateConversation(c *models.Conversation) error {
	return r.db.Create(c).Error
}

func (r *ChatRepository) GetConversationByID(id uuid.UUID) (*models.Conversation, error) {
	var conv models.Conversation
	err := r.db.First(&conv, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ChatRepository) GetOpenByUserID(userID uuid.UUID) (*models.Conversation, error) {
	var conv models.Conversation
	err := r.db.Where("user_id = ? AND status = ?", userID, models.ConversationStatusOpen).
		Order("created_at DESC").
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ChatRepository) GetOpenByGuestID(guestID uuid.UUID) (*models.Conversation, error) {
	var conv models.Conversation
	err := r.db.Where("guest_id = ? AND status = ?", guestID, models.ConversationStatusOpen).
		Order("created_at DESC").
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ChatRepository) UpdateConversation(c *models.Conversation) error {
	return r.db.Save(c).Error
}

func (r *ChatRepository) LinkGuestConversations(guestID, userID uuid.UUID) ([]uuid.UUID, error) {
	var convs []models.Conversation
	if err := r.db.Where("guest_id = ? AND user_id IS NULL", guestID).Find(&convs).Error; err != nil {
		return nil, err
	}
	if len(convs) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, len(convs))
	for i := range convs {
		ids[i] = convs[i].ID
	}
	err := r.db.Model(&models.Conversation{}).
		Where("guest_id = ? AND user_id IS NULL", guestID).
		Updates(map[string]interface{}{
			"user_id":  userID,
			"guest_id": nil,
		}).Error
	return ids, err
}

type AdminConversationListParams struct {
	Status     *models.ConversationStatus
	GuestOnly  bool
	UnreadOnly bool
	Page       int
	Limit      int
}

func (r *ChatRepository) ListAdmin(params AdminConversationListParams) ([]models.Conversation, int64, error) {
	page, limit := params.Page, params.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := r.db.Model(&models.Conversation{})
	if params.Status != nil {
		q = q.Where("status = ?", *params.Status)
	}
	if params.GuestOnly {
		q = q.Where("guest_id IS NOT NULL AND user_id IS NULL")
	}
	if params.UnreadOnly {
		q = q.Where(`EXISTS (
			SELECT 1 FROM chat_messages m
			WHERE m.conversation_id = conversations.id
			  AND m.sender_type = ?
			  AND (conversations.admin_last_read_at IS NULL OR m.created_at > conversations.admin_last_read_at)
		)`, models.MessageSenderCustomer)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.Conversation
	err := q.Order("last_message_at DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (r *ChatRepository) CountAdminUnreadConversations() (int64, error) {
	var count int64
	err := r.db.Model(&models.Conversation{}).
		Where("status = ?", models.ConversationStatusOpen).
		Where(`EXISTS (
			SELECT 1 FROM chat_messages m
			WHERE m.conversation_id = conversations.id
			  AND m.sender_type = ?
			  AND (conversations.admin_last_read_at IS NULL OR m.created_at > conversations.admin_last_read_at)
		)`, models.MessageSenderCustomer).
		Count(&count).Error
	return count, err
}

func (r *ChatRepository) CountUnreadForCustomer(conv *models.Conversation) (int64, error) {
	q := r.db.Model(&models.ChatMessage{}).
		Where("conversation_id = ? AND sender_type = ?", conv.ID, models.MessageSenderAdmin)
	if conv.CustomerLastReadAt != nil {
		q = q.Where("created_at > ?", *conv.CustomerLastReadAt)
	}
	var count int64
	err := q.Count(&count).Error
	return count, err
}

func (r *ChatRepository) CountUnreadForAdmin(conv *models.Conversation) (int64, error) {
	q := r.db.Model(&models.ChatMessage{}).
		Where("conversation_id = ? AND sender_type = ?", conv.ID, models.MessageSenderCustomer)
	if conv.AdminLastReadAt != nil {
		q = q.Where("created_at > ?", *conv.AdminLastReadAt)
	}
	var count int64
	err := q.Count(&count).Error
	return count, err
}

func (r *ChatRepository) CountMessages(conversationID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.ChatMessage{}).Where("conversation_id = ?", conversationID).Count(&count).Error
	return count, err
}

func (r *ChatRepository) CreateMessage(m *models.ChatMessage) error {
	return r.db.Create(m).Error
}

func (r *ChatRepository) ListMessages(conversationID uuid.UUID, page, limit int, since *time.Time) ([]models.ChatMessage, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	q := r.db.Model(&models.ChatMessage{}).Where("conversation_id = ?", conversationID)
	if since != nil {
		q = q.Where("created_at > ?", *since)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.ChatMessage
	err := q.Order("created_at ASC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (r *ChatRepository) TouchLastMessageAt(conversationID uuid.UUID, at time.Time) error {
	return r.db.Model(&models.Conversation{}).
		Where("id = ?", conversationID).
		Update("last_message_at", at).Error
}

func (r *ChatRepository) MarkCustomerRead(conversationID uuid.UUID, at time.Time) error {
	return r.db.Model(&models.Conversation{}).
		Where("id = ?", conversationID).
		Update("customer_last_read_at", at).Error
}

func (r *ChatRepository) MarkAdminRead(conversationID uuid.UUID, at time.Time) error {
	return r.db.Model(&models.Conversation{}).
		Where("id = ?", conversationID).
		Update("admin_last_read_at", at).Error
}
