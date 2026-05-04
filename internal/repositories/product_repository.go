package repositories

import (
	"auto-store-api/internal/models"
	"strings"

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

// List returns a page of products, optionally filtered by category slug, search string,
// and/or min/max price (inclusive on each bound when set).
func (r *ProductRepository) List(offset, limit int, categorySlug, search string, minPrice, maxPrice *float64) ([]models.Product, int64, error) {
	var products []models.Product
	var total int64
	q := r.db.Model(&models.Product{}).Preload("Images")
	if minPrice != nil {
		q = q.Where("price >= ?", *minPrice)
	}
	if maxPrice != nil {
		q = q.Where("price <= ?", *maxPrice)
	}
	if search != "" {
		term := "%" + strings.ToLower(search) + "%"
		q = q.Where("(LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(sku) LIKE ? OR LOWER(manufacturer_part_number) LIKE ?)",
			term, term, term, term)
	}
	if categorySlug != "" {
		var cat models.Category
		if err := r.db.Where("slug = ?", categorySlug).First(&cat).Error; err == nil {
			var productIDs []uuid.UUID
			r.db.Model(&models.ProductCategory{}).Where("category_id = ?", cat.ID).Pluck("product_id", &productIDs)
			if len(productIDs) > 0 {
				q = q.Where("id IN ?", productIDs)
			} else {
				q = q.Where("1 = 0")
			}
		}
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&products).Error
	return products, total, err
}

func (r *ProductRepository) ListByIDs(ids []uuid.UUID, offset, limit int) ([]models.Product, int64, error) {
	if len(ids) == 0 {
		return []models.Product{}, 0, nil
	}
	var products []models.Product
	total := int64(len(ids))
	err := r.db.Preload("Images").Where("id IN ?", ids).Offset(offset).Limit(limit).Find(&products).Error
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

// DeleteProductImageByID soft-deletes an image if it belongs to the product.
func (r *ProductRepository) DeleteProductImageByID(productID, imageID uuid.UUID) error {
	var img models.ProductImage
	if err := r.db.Where("id = ? AND product_id = ?", imageID, productID).First(&img).Error; err != nil {
		return err
	}
	return r.db.Delete(&img).Error
}

// ReplaceProductImages deletes existing images for the product and inserts rows (transactional).
func (r *ProductRepository) ReplaceProductImages(productID uuid.UUID, images []models.ProductImage) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("product_id = ?", productID).Delete(&models.ProductImage{}).Error; err != nil {
			return err
		}
		for i := range images {
			images[i].ID = uuid.Nil
			images[i].ProductID = productID
			if err := tx.Create(&images[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
