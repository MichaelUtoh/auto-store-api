package repositories

import (
	"auto-store-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AddressRepository struct {
	db *gorm.DB
}

func NewAddressRepository(db *gorm.DB) *AddressRepository {
	return &AddressRepository{db: db}
}

func (r *AddressRepository) Create(a *models.Address) error {
	return r.db.Create(a).Error
}

func (r *AddressRepository) GetByID(id uuid.UUID) (*models.Address, error) {
	var a models.Address
	err := r.db.First(&a, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AddressRepository) GetByUserID(userID uuid.UUID) ([]models.Address, error) {
	var addrs []models.Address
	err := r.db.Where("user_id = ?", userID).Find(&addrs).Error
	return addrs, err
}

func (r *AddressRepository) GetByIDAndUser(id, userID uuid.UUID) (*models.Address, error) {
	var a models.Address
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AddressRepository) Update(a *models.Address) error {
	return r.db.Save(a).Error
}

func (r *AddressRepository) Delete(id, userID uuid.UUID) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Address{}).Error
}
