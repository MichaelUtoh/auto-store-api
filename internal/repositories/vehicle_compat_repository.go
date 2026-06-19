package repositories

import (
	"auto-store-api/internal/models"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VehicleCompatibilityRepository struct {
	db *gorm.DB
}

func NewVehicleCompatibilityRepository(db *gorm.DB) *VehicleCompatibilityRepository {
	return &VehicleCompatibilityRepository{db: db}
}

type ListVehicleCompatibilityFilter struct {
	Make          string
	Model         string
	MarketVariant string
	Page          int
	Limit         int
}

// LinkedCompatibility is a catalog entry plus optional per-product link notes.
type LinkedCompatibility struct {
	models.VehicleCompatibility
	LinkNotes string `json:"link_notes,omitempty"`
}

func (r *VehicleCompatibilityRepository) List(filter ListVehicleCompatibilityFilter) ([]models.VehicleCompatibility, int64, error) {
	q := r.db.Model(&models.VehicleCompatibility{})
	if filter.Make != "" {
		q = q.Where("LOWER(make) = ?", strings.ToLower(filter.Make))
	}
	if filter.Model != "" {
		q = q.Where("LOWER(model) = ?", strings.ToLower(filter.Model))
	}
	if filter.MarketVariant != "" {
		q = q.Where("LOWER(market_variant) = ?", strings.ToLower(filter.MarketVariant))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	offset := (filter.Page - 1) * filter.Limit
	var list []models.VehicleCompatibility
	err := q.Order("make ASC, model ASC, year_start ASC").
		Offset(offset).Limit(filter.Limit).Find(&list).Error
	return list, total, err
}

func (r *VehicleCompatibilityRepository) GetByID(id uuid.UUID) (*models.VehicleCompatibility, error) {
	var v models.VehicleCompatibility
	if err := r.db.First(&v, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VehicleCompatibilityRepository) FindOrCreate(attrs models.VehicleCompatibility) (*models.VehicleCompatibility, error) {
	if attrs.MarketVariant == "" {
		attrs.MarketVariant = "nigeria"
	}
	var existing models.VehicleCompatibility
	err := r.db.Where(
		`LOWER(make) = ? AND LOWER(model) = ? AND generation = ? AND year_start = ? AND year_end = ?
		 AND LOWER(engine) = ? AND trim = ? AND LOWER(market_variant) = ?`,
		strings.ToLower(attrs.Make),
		strings.ToLower(attrs.Model),
		attrs.Generation,
		attrs.YearStart,
		attrs.YearEnd,
		strings.ToLower(attrs.Engine),
		attrs.Trim,
		strings.ToLower(attrs.MarketVariant),
	).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err := r.db.Create(&attrs).Error; err != nil {
		return nil, err
	}
	return &attrs, nil
}

func (r *VehicleCompatibilityRepository) Create(attrs *models.VehicleCompatibility) error {
	created, err := r.FindOrCreate(*attrs)
	if err != nil {
		return err
	}
	*attrs = *created
	return nil
}

func (r *VehicleCompatibilityRepository) LinkProduct(productID uuid.UUID, compatibilityID uuid.UUID, notes string) error {
	link := models.ProductVehicleCompatibility{
		ProductID:              productID,
		VehicleCompatibilityID: compatibilityID,
		Notes:                  notes,
	}
	var existing models.ProductVehicleCompatibility
	err := r.db.Where("product_id = ? AND vehicle_compatibility_id = ?", productID, compatibilityID).
		First(&existing).Error
	if err == nil {
		if notes != "" && existing.Notes != notes {
			return r.db.Model(&models.ProductVehicleCompatibility{}).
				Where("product_id = ? AND vehicle_compatibility_id = ?", productID, compatibilityID).
				Update("notes", notes).Error
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return r.db.Create(&link).Error
}

func (r *VehicleCompatibilityRepository) GetLinkedByProductID(productID uuid.UUID) ([]LinkedCompatibility, error) {
	var rows []struct {
		models.VehicleCompatibility
		LinkNotes string `gorm:"column:link_notes"`
		LinkedAt  time.Time
	}
	err := r.db.Table("vehicle_compatibilities").
		Select("vehicle_compatibilities.*, product_vehicle_compatibilities.notes AS link_notes").
		Joins("JOIN product_vehicle_compatibilities ON product_vehicle_compatibilities.vehicle_compatibility_id = vehicle_compatibilities.id").
		Where("product_vehicle_compatibilities.product_id = ?", productID).
		Order("vehicle_compatibilities.make ASC, vehicle_compatibilities.model ASC, vehicle_compatibilities.year_start ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]LinkedCompatibility, len(rows))
	for i, row := range rows {
		out[i] = LinkedCompatibility{
			VehicleCompatibility: row.VehicleCompatibility,
			LinkNotes:            row.LinkNotes,
		}
	}
	return out, nil
}

func (r *VehicleCompatibilityRepository) DeleteLinksByProductID(productID uuid.UUID) error {
	return r.db.Where("product_id = ?", productID).Delete(&models.ProductVehicleCompatibility{}).Error
}

func applyVehicleCompatibilityFilter(db *gorm.DB, make, model string, yearStart, yearEnd *int, trim, engine string) *gorm.DB {
	sub := db.Model(&models.ProductVehicleCompatibility{}).
		Select("product_vehicle_compatibilities.product_id").
		Joins("JOIN vehicle_compatibilities ON vehicle_compatibilities.id = product_vehicle_compatibilities.vehicle_compatibility_id")
	if make != "" {
		sub = sub.Where("LOWER(vehicle_compatibilities.make) = ?", strings.ToLower(make))
	}
	if model != "" {
		sub = sub.Where("LOWER(vehicle_compatibilities.model) = ?", strings.ToLower(model))
	}
	if yearStart != nil {
		sub = sub.Where("vehicle_compatibilities.year_end = 0 OR vehicle_compatibilities.year_end >= ?", *yearStart)
	}
	if yearEnd != nil {
		sub = sub.Where("vehicle_compatibilities.year_start = 0 OR vehicle_compatibilities.year_start <= ?", *yearEnd)
	}
	if trim != "" {
		sub = sub.Where("(vehicle_compatibilities.trim = '' OR LOWER(vehicle_compatibilities.trim) = ?)", strings.ToLower(trim))
	}
	if engine != "" {
		sub = sub.Where("(vehicle_compatibilities.engine = '' OR LOWER(vehicle_compatibilities.engine) = ?)", strings.ToLower(engine))
	}
	return sub
}
