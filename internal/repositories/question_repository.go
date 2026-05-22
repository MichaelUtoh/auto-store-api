package repositories

import (
	"auto-store-api/internal/models"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QuestionListFilter struct {
	Query      string
	ProductID  *uuid.UUID
	CategoryID *uuid.UUID
	Make       string
	Model      string
	Year       *int
	Status     *models.QuestionStatus
	ExcludeClosed bool
}

type QuestionRepository struct {
	db *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

func (r *QuestionRepository) Create(q *models.Question) error {
	return r.db.Create(q).Error
}

func (r *QuestionRepository) CreateAnswer(a *models.Answer) error {
	return r.db.Create(a).Error
}

func (r *QuestionRepository) UpdateQuestion(q *models.Question) error {
	return r.db.Save(q).Error
}

func (r *QuestionRepository) UpdateAnswer(a *models.Answer) error {
	return r.db.Save(a).Error
}

func (r *QuestionRepository) SlugExists(slug string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Question{}).Where("slug = ?", slug).Count(&count).Error
	return count > 0, err
}

func (r *QuestionRepository) GetByID(id uuid.UUID) (*models.Question, error) {
	var q models.Question
	err := r.db.Preload("User").Preload("Product").Preload("Category").
		Preload("Answers", func(db *gorm.DB) *gorm.DB {
			return db.Order("is_accepted DESC, created_at ASC")
		}).
		Preload("Answers.User.MechanicProfile").
		First(&q, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *QuestionRepository) GetBySlug(slug string) (*models.Question, error) {
	var q models.Question
	err := r.db.Preload("User").Preload("Product").Preload("Category").
		Preload("Answers", func(db *gorm.DB) *gorm.DB {
			return db.Order("is_accepted DESC, created_at ASC")
		}).
		Preload("Answers.User.MechanicProfile").
		First(&q, "slug = ?", slug).Error
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *QuestionRepository) IncrementViewCount(id uuid.UUID) error {
	return r.db.Model(&models.Question{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

func (r *QuestionRepository) List(filter QuestionListFilter, page, limit int) ([]models.Question, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := r.db.Model(&models.Question{})
	if filter.ExcludeClosed {
		q = q.Where("status != ?", models.QuestionStatusClosed)
	}
	if filter.Status != nil {
		q = q.Where("status = ?", *filter.Status)
	}
	if filter.ProductID != nil {
		q = q.Where("product_id = ?", *filter.ProductID)
	}
	if filter.CategoryID != nil {
		q = q.Where("category_id = ?", *filter.CategoryID)
	}
	if filter.Make != "" {
		q = q.Where("LOWER(make) = ?", strings.ToLower(filter.Make))
	}
	if filter.Model != "" {
		q = q.Where("LOWER(model) = ?", strings.ToLower(filter.Model))
	}
	if filter.Year != nil {
		q = q.Where("year = ?", *filter.Year)
	}
	if filter.Query != "" {
		like := "%" + strings.ToLower(filter.Query) + "%"
		q = q.Where("LOWER(title) LIKE ? OR LOWER(body) LIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var questions []models.Question
	err := q.Preload("User").
		Preload("Answers", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_accepted = ?", true).Limit(1)
		}).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&questions).Error
	return questions, total, err
}

func (r *QuestionRepository) GetAnswerByID(answerID, questionID uuid.UUID) (*models.Answer, error) {
	var a models.Answer
	err := r.db.First(&a, "id = ? AND question_id = ?", answerID, questionID).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *QuestionRepository) ClearAcceptedAnswers(questionID uuid.UUID) error {
	return r.db.Model(&models.Answer{}).
		Where("question_id = ?", questionID).
		Update("is_accepted", false).Error
}

func (r *QuestionRepository) CountAnswers(questionID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.Answer{}).Where("question_id = ?", questionID).Count(&count).Error
	return count, err
}
