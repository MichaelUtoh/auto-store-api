package services

import (
	"context"
	"errors"
	"fmt"

	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"auto-store-api/internal/validators"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrQuestionNotFound      = errors.New("question not found")
	ErrAnswerNotFound          = errors.New("answer not found")
	ErrQuestionClosed          = errors.New("question is closed")
	ErrNotQuestionAuthor       = errors.New("only the question author can perform this action")
	ErrInvalidQuestionContext  = errors.New("provide at least one of product_id, category_id, or vehicle make/model")
	ErrCategoryNotFound        = errors.New("category not found")
	ErrAlreadyAnsweredByUser   = errors.New("you have already answered this question")
)

type CreateQuestionInput struct {
	Title      string
	Body       string
	ProductID  *uuid.UUID
	CategoryID *uuid.UUID
	Make       string
	Model      string
	Year       *int
}

type QuestionListParams struct {
	Query      string
	ProductID  *uuid.UUID
	CategoryID *uuid.UUID
	Make       string
	Model      string
	Year       *int
	Status     *models.QuestionStatus
	Page       int
	Limit      int
}

type QuestionService struct {
	questionRepo *repositories.QuestionRepository
	productRepo  *repositories.ProductRepository
	categoryRepo *repositories.CategoryRepository
	notifier     *Notifier
	db           *gorm.DB
}

func NewQuestionService(
	questionRepo *repositories.QuestionRepository,
	productRepo *repositories.ProductRepository,
	categoryRepo *repositories.CategoryRepository,
	notifier *Notifier,
	db *gorm.DB,
) *QuestionService {
	return &QuestionService{
		questionRepo: questionRepo,
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
		notifier:     notifier,
		db:           db,
	}
}

func (s *QuestionService) Create(userID uuid.UUID, input CreateQuestionInput) (*models.Question, error) {
	if err := s.validateQuestionContext(input); err != nil {
		return nil, err
	}

	slug, err := s.uniqueSlug(input.Title)
	if err != nil {
		return nil, err
	}

	q := &models.Question{
		UserID:     userID,
		Title:      input.Title,
		Body:       input.Body,
		Slug:       slug,
		ProductID:  input.ProductID,
		CategoryID: input.CategoryID,
		Make:       input.Make,
		Model:      input.Model,
		Year:       input.Year,
		Status:     models.QuestionStatusOpen,
	}
	if err := s.questionRepo.Create(q); err != nil {
		return nil, err
	}
	return s.questionRepo.GetByID(q.ID)
}

func (s *QuestionService) List(params QuestionListParams) ([]models.Question, int64, error) {
	filter := repositories.QuestionListFilter{
		Query:         params.Query,
		ProductID:     params.ProductID,
		CategoryID:    params.CategoryID,
		Make:          params.Make,
		Model:         params.Model,
		Year:          params.Year,
		Status:        params.Status,
		ExcludeClosed: params.Status == nil,
	}
	questions, total, err := s.questionRepo.List(filter, params.Page, params.Limit)
	if err != nil {
		return nil, 0, err
	}
	for i := range questions {
		if count, err := s.questionRepo.CountAnswers(questions[i].ID); err == nil {
			questions[i].AnswerCount = int(count)
		}
	}
	return questions, total, nil
}

func (s *QuestionService) GetBySlug(slug string, trackView bool) (*models.Question, error) {
	q, err := s.questionRepo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}
	if q.Status == models.QuestionStatusClosed {
		return nil, ErrQuestionNotFound
	}
	if trackView {
		_ = s.questionRepo.IncrementViewCount(q.ID)
		q.ViewCount++
	}
	return q, nil
}

func (s *QuestionService) GetByID(id uuid.UUID) (*models.Question, error) {
	q, err := s.questionRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}
	return q, nil
}

func (s *QuestionService) PostAnswer(ctx context.Context, questionID, mechanicUserID uuid.UUID, body string, verified bool) (*models.Answer, error) {
	q, err := s.GetByID(questionID)
	if err != nil {
		return nil, err
	}
	if q.Status == models.QuestionStatusClosed {
		return nil, ErrQuestionClosed
	}

	var existing int64
	if err := s.db.Model(&models.Answer{}).
		Where("question_id = ? AND user_id = ?", questionID, mechanicUserID).
		Count(&existing).Error; err != nil {
		return nil, err
	}
	if existing > 0 {
		return nil, ErrAlreadyAnsweredByUser
	}

	answer := &models.Answer{
		QuestionID:         questionID,
		UserID:             mechanicUserID,
		Body:               body,
		IsVerifiedMechanic: verified,
	}

	var result *models.Answer
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(answer).Error; err != nil {
			return err
		}
		if q.Status == models.QuestionStatusOpen {
			q.Status = models.QuestionStatusAnswered
			if err := tx.Save(q).Error; err != nil {
				return err
			}
		}
		result = answer
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.notifier != nil && q.UserID != mechanicUserID {
		_ = s.notifier.QAAnswerPosted(ctx, q.UserID, q.ID, q.Slug)
	}

	full, err := s.questionRepo.GetByID(questionID)
	if err != nil {
		return result, nil
	}
	for i := range full.Answers {
		if full.Answers[i].ID == result.ID {
			return &full.Answers[i], nil
		}
	}
	return result, nil
}

func (s *QuestionService) AcceptAnswer(questionID, answerID, authorUserID uuid.UUID) (*models.Question, error) {
	q, err := s.GetByID(questionID)
	if err != nil {
		return nil, err
	}
	if q.UserID != authorUserID {
		return nil, ErrNotQuestionAuthor
	}
	if q.Status == models.QuestionStatusClosed {
		return nil, ErrQuestionClosed
	}

	answer, err := s.questionRepo.GetAnswerByID(answerID, questionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAnswerNotFound
		}
		return nil, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Answer{}).
			Where("question_id = ?", questionID).
			Update("is_accepted", false).Error; err != nil {
			return err
		}
		answer.IsAccepted = true
		if err := tx.Save(answer).Error; err != nil {
			return err
		}
		q.Status = models.QuestionStatusAnswered
		return tx.Save(q).Error
	})
	if err != nil {
		return nil, err
	}
	return s.questionRepo.GetByID(questionID)
}

func (s *QuestionService) Close(questionID uuid.UUID, actorUserID uuid.UUID, isAdmin bool) (*models.Question, error) {
	q, err := s.GetByID(questionID)
	if err != nil {
		return nil, err
	}
	if !isAdmin && q.UserID != actorUserID {
		return nil, ErrNotQuestionAuthor
	}
	q.Status = models.QuestionStatusClosed
	if err := s.questionRepo.UpdateQuestion(q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *QuestionService) validateQuestionContext(input CreateQuestionInput) error {
	hasProduct := input.ProductID != nil
	hasCategory := input.CategoryID != nil
	hasVehicle := input.Make != "" && input.Model != ""
	if !hasProduct && !hasCategory && !hasVehicle {
		return ErrInvalidQuestionContext
	}
	if hasProduct {
		if _, err := s.productRepo.GetByID(*input.ProductID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProductNotFound
			}
			return err
		}
	}
	if hasCategory {
		if _, err := s.categoryRepo.GetByID(*input.CategoryID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCategoryNotFound
			}
			return err
		}
	}
	return nil
}

func (s *QuestionService) uniqueSlug(title string) (string, error) {
	base := validators.Slugify(title)
	if base == "" {
		base = "question"
	}
	if len(base) > 80 {
		base = base[:80]
		base = stringsTrimSuffixDash(base)
	}
	slug := base
	for i := 0; i < 10; i++ {
		exists, err := s.questionRepo.SlugExists(slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		suffix := uuid.New().String()[:8]
		slug = fmt.Sprintf("%s-%s", base, suffix)
	}
	return "", errors.New("could not generate unique slug")
}

func stringsTrimSuffixDash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '-' {
		s = s[:len(s)-1]
	}
	return s
}
