package services

import (
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"auto-store-api/internal/validators"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryService struct {
	repo *repositories.CategoryRepository
	db   *gorm.DB
}

func NewCategoryService(repo *repositories.CategoryRepository, db *gorm.DB) *CategoryService {
	return &CategoryService{repo: repo, db: db}
}

func (s *CategoryService) Create(name, slug, description string, parentID *uuid.UUID) (*models.Category, error) {
	if slug == "" {
		slug = validators.Slugify(name)
	}
	level := 0
	if parentID != nil {
		parent, err := s.repo.GetByID(*parentID)
		if err == nil {
			level = parent.Level + 1
		}
	}
	c := &models.Category{
		ParentID:    parentID,
		Name:        name,
		Slug:        slug,
		Description: description,
		Level:       level,
	}
	if err := s.repo.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CategoryService) GetByID(id uuid.UUID) (*models.Category, error) {
	return s.repo.GetByID(id)
}

func (s *CategoryService) GetBySlug(slug string) (*models.Category, error) {
	return s.repo.GetBySlug(slug)
}

func (s *CategoryService) ListTree() ([]models.Category, error) {
	return s.repo.ListTree()
}

func (s *CategoryService) Update(c *models.Category) error {
	return s.repo.Update(c)
}

func (s *CategoryService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}

func (s *CategoryService) GetCategoryProductIDs(categoryID uuid.UUID, includeChildren bool) ([]uuid.UUID, error) {
	return s.repo.GetCategoryProductIDs(categoryID, includeChildren)
}
