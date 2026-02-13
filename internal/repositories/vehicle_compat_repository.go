package repositories

import (
	"auto-store-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VehicleCompatibilityRepository struct {
	db *gorm.DB
}

func NewVehicleCompatibilityRepository(db *gorm.DB) *VehicleCompatibilityRepository {
	return &VehicleCompatibilityRepository{db: db}
}

func (r *VehicleCompatibilityRepository) GetByProductID(productID uuid.UUID) ([]models.VehicleCompatibility, error) {
	var list []models.VehicleCompatibility
	err := r.db.Where("product_id = ?", productID).Find(&list).Error
	return list, err
}

func (r *VehicleCompatibilityRepository) Create(v *models.VehicleCompatibility) error {
	return r.db.Create(v).Error
}

func (r *VehicleCompatibilityRepository) DeleteByProductID(productID uuid.UUID) error {
	return r.db.Where("product_id = ?", productID).Delete(&models.VehicleCompatibility{}).Error
}
