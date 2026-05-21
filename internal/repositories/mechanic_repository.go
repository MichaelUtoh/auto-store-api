package repositories

import (
	"auto-store-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MechanicRepository struct {
	db *gorm.DB
}

func NewMechanicRepository(db *gorm.DB) *MechanicRepository {
	return &MechanicRepository{db: db}
}

func (r *MechanicRepository) CreateProfile(profile *models.MechanicProfile) error {
	return r.db.Create(profile).Error
}

func (r *MechanicRepository) UpdateProfile(profile *models.MechanicProfile) error {
	return r.db.Save(profile).Error
}

func (r *MechanicRepository) GetProfileByUserID(userID uuid.UUID) (*models.MechanicProfile, error) {
	var profile models.MechanicProfile
	err := r.db.Preload("Documents").First(&profile, "user_id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *MechanicRepository) GetProfileByID(id uuid.UUID) (*models.MechanicProfile, error) {
	var profile models.MechanicProfile
	err := r.db.Preload("Documents").Preload("User").First(&profile, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *MechanicRepository) ProfileExistsForUser(userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.MechanicProfile{}).Where("user_id = ?", userID).Count(&count).Error
	return count > 0, err
}

func (r *MechanicRepository) ListProfiles(status *models.MechanicProfileStatus, page, limit int) ([]models.MechanicProfile, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := r.db.Model(&models.MechanicProfile{})
	if status != nil {
		q = q.Where("status = ?", *status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var profiles []models.MechanicProfile
	err := q.Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&profiles).Error
	return profiles, total, err
}

func (r *MechanicRepository) ListVerified(page, limit int) ([]models.MechanicProfile, int64, error) {
	status := models.MechanicStatusVerified
	return r.ListProfiles(&status, page, limit)
}

func (r *MechanicRepository) CreateDocument(doc *models.MechanicDocument) error {
	return r.db.Create(doc).Error
}

func (r *MechanicRepository) GetDocumentByID(id, profileID uuid.UUID) (*models.MechanicDocument, error) {
	var doc models.MechanicDocument
	err := r.db.First(&doc, "id = ? AND mechanic_profile_id = ?", id, profileID).Error
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *MechanicRepository) DeleteDocument(id, profileID uuid.UUID) error {
	return r.db.Where("id = ? AND mechanic_profile_id = ?", id, profileID).Delete(&models.MechanicDocument{}).Error
}
