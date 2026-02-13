package repositories

import (
	"auto-store-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(c *models.Category) error {
	return r.db.Create(c).Error
}

func (r *CategoryRepository) GetByID(id uuid.UUID) (*models.Category, error) {
	var c models.Category
	err := r.db.Preload("Children").First(&c, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) GetBySlug(slug string) (*models.Category, error) {
	var c models.Category
	err := r.db.First(&c, "slug = ?", slug).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) Update(c *models.Category) error {
	return r.db.Save(c).Error
}

func (r *CategoryRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Category{}, "id = ?", id).Error
}

func (r *CategoryRepository) ListTree() ([]models.Category, error) {
	var categories []models.Category
	err := r.db.Where("parent_id IS NULL").Preload("Children").Find(&categories).Error
	return categories, err
}

func (r *CategoryRepository) GetCategoryProductIDs(categoryID uuid.UUID, includeChildren bool) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	q := r.db.Model(&models.ProductCategory{}).Where("category_id = ?", categoryID).Pluck("product_id", &ids)
	if err := q.Error; err != nil {
		return nil, err
	}
	if includeChildren {
		var childIDs []uuid.UUID
		r.db.Model(&models.Category{}).Where("parent_id = ?", categoryID).Pluck("id", &childIDs)
		for _, cid := range childIDs {
			var more []uuid.UUID
			r.db.Model(&models.ProductCategory{}).Where("category_id = ?", cid).Pluck("product_id", &more)
			ids = append(ids, more...)
		}
	}
	return ids, nil
}
