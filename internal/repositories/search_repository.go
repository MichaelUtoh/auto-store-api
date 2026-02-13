package repositories

import (
	"auto-store-api/internal/models"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type SearchParams struct {
	Q         string
	Category  string
	Tags      []string
	Make      string
	Model     string
	YearStart *int
	YearEnd   *int
	MinPrice  *float64
	MaxPrice  *float64
	Condition string
	Brand     string
	Sort      string
	Page      int
	Limit     int
}

type SearchResult struct {
	Products []models.Product
	Total    int64
}

func (r *ProductRepository) Search(params SearchParams) (*SearchResult, error) {
	q := r.db.Model(&models.Product{})

	if params.Q != "" {
		term := "%" + strings.ToLower(params.Q) + "%"
		q = q.Where("(LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(sku) LIKE ? OR LOWER(manufacturer_part_number) LIKE ?)",
			term, term, term, term)
	}
	if params.Brand != "" {
		q = q.Where("LOWER(brand) = ?", strings.ToLower(params.Brand))
	}
	if params.MinPrice != nil {
		q = q.Where("price >= ?", *params.MinPrice)
	}
	if params.MaxPrice != nil {
		q = q.Where("price <= ?", *params.MaxPrice)
	}
	if params.Condition != "" {
		q = q.Where("condition = ?", params.Condition)
	}
	if params.Category != "" {
		var cat models.Category
		if err := r.db.Where("slug = ?", params.Category).First(&cat).Error; err == nil {
			var productIDs []uuid.UUID
			r.db.Model(&models.ProductCategory{}).Where("category_id = ?", cat.ID).Pluck("product_id", &productIDs)
			if len(productIDs) > 0 {
				q = q.Where("id IN ?", productIDs)
			} else {
				q = q.Where("1 = 0")
			}
		}
	}
	if len(params.Tags) > 0 {
		sub := r.db.Model(&models.ProductTag{}).Select("product_id").
			Joins("JOIN tags ON tags.id = product_tags.tag_id").
			Where("LOWER(tags.slug) IN ?", toLower(params.Tags)).
			Group("product_id").
			Having("COUNT(DISTINCT tags.id) = ?", len(params.Tags))
		q = q.Where("id IN (?)", sub)
	}
	if params.Make != "" || params.Model != "" || params.YearStart != nil || params.YearEnd != nil {
		compatSub := r.db.Model(&models.VehicleCompatibility{})
		if params.Make != "" {
			compatSub = compatSub.Where("LOWER(make) = ?", strings.ToLower(params.Make))
		}
		if params.Model != "" {
			compatSub = compatSub.Where("LOWER(model) = ?", strings.ToLower(params.Model))
		}
		if params.YearStart != nil {
			compatSub = compatSub.Where("year_end >= ?", *params.YearStart)
		}
		if params.YearEnd != nil {
			compatSub = compatSub.Where("year_start <= ?", *params.YearEnd)
		}
		var productIDs []uuid.UUID
		compatSub.Pluck("product_id", &productIDs)
		if len(productIDs) > 0 {
			q = q.Where("id IN ?", productIDs)
		} else if params.Make != "" || params.Model != "" {
			q = q.Where("1 = 0")
		}
	}

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return nil, err
	}

	switch params.Sort {
	case "price_asc":
		q = q.Order("price ASC")
	case "price_desc":
		q = q.Order("price DESC")
	case "newest":
		q = q.Order("created_at DESC")
	case "rating", "popularity":
		q = q.Order("created_at DESC")
	default:
		q = q.Order("created_at DESC")
	}

	offset := (params.Page - 1) * params.Limit
	if offset < 0 {
		offset = 0
	}
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	var products []models.Product
	err := q.Preload("Images").Offset(offset).Limit(params.Limit).Find(&products).Error
	if err != nil {
		return nil, err
	}
	return &SearchResult{Products: products, Total: count}, nil
}

func toLower(s []string) []string {
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = strings.ToLower(v)
	}
	return out
}

// ParseYearRange parses "2015-2020" into start and end
func ParseYearRange(s string) (start, end *int) {
	if s == "" {
		return nil, nil
	}
	var a, b int
	if _, err := fmt.Sscanf(s, "%d-%d", &a, &b); err == nil {
		start, end = &a, &b
		return
	}
	if _, err := fmt.Sscanf(s, "%d", &a); err == nil {
		start = &a
		end = &a
	}
	return
}
