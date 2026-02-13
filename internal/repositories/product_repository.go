package repositories

import (
	"auto-store-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(p *models.Product) error {
	return r.db.Create(p).Error
}

func (r *ProductRepository) GetByID(id uuid.UUID, preload ...string) (*models.Product, error) {
	var p models.Product
	q := r.db.Model(&models.Product{})
	for _, rel := range preload {
		q = q.Preload(rel)
	}
	err := q.First(&p, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepository) GetBySKU(sku string) (*models.Product, error) {
	var p models.Product
	err := r.db.First(&p, "sku = ?", sku).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepository) Update(p *models.Product) error {
	return r.db.Save(p).Error
}

func (r *ProductRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Product{}, "id = ?", id).Error
}

func (r *ProductRepository) List(offset, limit int, filters map[string]interface{}) ([]models.Product, int64, error) {
	var products []models.Product
	var total int64
	q := r.db.Model(&models.Product{})
	for k, v := range filters {
		if v != nil && v != "" && v != 0 {
			q = q.Where(k+" = ?", v)
		}
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Offset(offset).Limit(limit).Find(&products).Error
	return products, total, err
}

func (r *ProductRepository) ListByIDs(ids []uuid.UUID, offset, limit int) ([]models.Product, int64, error) {
	if len(ids) == 0 {
		return []models.Product{}, 0, nil
	}
	var products []models.Product
	total := int64(len(ids))
	err := r.db.Where("id IN ?", ids).Offset(offset).Limit(limit).Find(&products).Error
	return products, total, err
}

func (r *ProductRepository) SetCategories(productID uuid.UUID, categoryIDs []uuid.UUID) error {
	if err := r.db.Where("product_id = ?", productID).Delete(&models.ProductCategory{}).Error; err != nil {
		return err
	}
	for _, cid := range categoryIDs {
		if err := r.db.Create(&models.ProductCategory{ProductID: productID, CategoryID: cid}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *ProductRepository) SetTags(productID uuid.UUID, tagIDs []uuid.UUID) error {
	if err := r.db.Where("product_id = ?", productID).Delete(&models.ProductTag{}).Error; err != nil {
		return err
	}
	for _, tid := range tagIDs {
		if err := r.db.Create(&models.ProductTag{ProductID: productID, TagID: tid}).Error; err != nil {
			return err
		}
	}
	return nil
}

// CreateProductImage inserts a product image.
func (r *ProductRepository) CreateProductImage(img *models.ProductImage) error {
	return r.db.Create(img).Error
}

// GetProductImagesByProductID returns all images for a product ordered by display_order.
func (r *ProductRepository) GetProductImagesByProductID(productID uuid.UUID) ([]models.ProductImage, error) {
	var images []models.ProductImage
	err := r.db.Where("product_id = ?", productID).Order("display_order ASC, created_at ASC").Find(&images).Error
	return images, err
}

// UnsetPrimaryImages sets is_primary = false for all images of a product.
func (r *ProductRepository) UnsetPrimaryImages(productID uuid.UUID) error {
	return r.db.Model(&models.ProductImage{}).Where("product_id = ?", productID).Update("is_primary", false).Error
}
