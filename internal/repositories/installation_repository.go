package repositories

import (
	"auto-store-api/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InstallationRepository struct {
	db *gorm.DB
}

func NewInstallationRepository(db *gorm.DB) *InstallationRepository {
	return &InstallationRepository{db: db}
}

func (r *InstallationRepository) ListJobTypes(activeOnly bool) ([]models.InstallationJobType, error) {
	q := r.db.Model(&models.InstallationJobType{})
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	var types []models.InstallationJobType
	err := q.Order("name ASC").Find(&types).Error
	return types, err
}

func (r *InstallationRepository) GetJobTypeByID(id uuid.UUID) (*models.InstallationJobType, error) {
	var jt models.InstallationJobType
	err := r.db.First(&jt, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &jt, nil
}

func (r *InstallationRepository) GetJobTypeByCode(code string) (*models.InstallationJobType, error) {
	var jt models.InstallationJobType
	err := r.db.First(&jt, "code = ?", code).Error
	if err != nil {
		return nil, err
	}
	return &jt, nil
}

func (r *InstallationRepository) ListVerifiedMechanicsWithCoords() ([]models.MechanicProfile, error) {
	var profiles []models.MechanicProfile
	err := r.db.Where("status = ? AND latitude IS NOT NULL AND longitude IS NOT NULL", models.MechanicStatusVerified).
		Find(&profiles).Error
	return profiles, err
}

func (r *InstallationRepository) ListInstallServicesForMechanic(mechanicProfileID uuid.UUID) ([]models.MechanicInstallService, error) {
	var services []models.MechanicInstallService
	err := r.db.Where("mechanic_profile_id = ? AND is_active = ?", mechanicProfileID, true).
		Preload("JobType").
		Find(&services).Error
	return services, err
}

func (r *InstallationRepository) MechanicOffersJobType(mechanicProfileID, jobTypeID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.MechanicInstallService{}).
		Where("mechanic_profile_id = ? AND job_type_id = ? AND is_active = ?", mechanicProfileID, jobTypeID, true).
		Count(&count).Error
	return count > 0, err
}

func (r *InstallationRepository) UpsertInstallService(s *models.MechanicInstallService) error {
	var existing models.MechanicInstallService
	err := r.db.Where("mechanic_profile_id = ? AND job_type_id = ?", s.MechanicProfileID, s.JobTypeID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(s).Error
	}
	if err != nil {
		return err
	}
	existing.LaborPrice = s.LaborPrice
	existing.IsActive = s.IsActive
	return r.db.Save(&existing).Error
}

func (r *InstallationRepository) CreateQuote(quote *models.InstallationQuote) error {
	return r.db.Create(quote).Error
}

func (r *InstallationRepository) CreateQuoteItems(items []models.InstallationQuoteItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.Create(&items).Error
}

func (r *InstallationRepository) CreateQuoteLines(lines []models.InstallationQuoteLine) error {
	if len(lines) == 0 {
		return nil
	}
	return r.db.Create(&lines).Error
}

func (r *InstallationRepository) GetQuoteByID(id uuid.UUID) (*models.InstallationQuote, error) {
	var quote models.InstallationQuote
	err := r.db.Preload("Lines.MechanicProfile").
		Preload("Lines.JobType").
		Preload("Items.Product").
		Preload("Items.JobType").
		First(&quote, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &quote, nil
}

func (r *InstallationRepository) GetQuoteByIDForUser(id, userID uuid.UUID) (*models.InstallationQuote, error) {
	var quote models.InstallationQuote
	err := r.db.Preload("Lines.MechanicProfile").
		Preload("Lines.JobType").
		Preload("Items.Product").
		Preload("Items.JobType").
		First(&quote, "id = ? AND user_id = ?", id, userID).Error
	if err != nil {
		return nil, err
	}
	return &quote, nil
}

func (r *InstallationRepository) UpdateQuote(quote *models.InstallationQuote) error {
	return r.db.Save(quote).Error
}

func (r *InstallationRepository) GetQuoteLineByID(id uuid.UUID) (*models.InstallationQuoteLine, error) {
	var line models.InstallationQuoteLine
	err := r.db.Preload("MechanicProfile").Preload("JobType").First(&line, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &line, nil
}

func (r *InstallationRepository) UpdateQuoteLine(line *models.InstallationQuoteLine) error {
	return r.db.Save(line).Error
}

func (r *InstallationRepository) ListQuoteLinesForMechanic(mechanicProfileID uuid.UUID, page, limit int) ([]models.InstallationQuoteLine, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := r.db.Model(&models.InstallationQuoteLine{}).Where("mechanic_profile_id = ?", mechanicProfileID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var lines []models.InstallationQuoteLine
	err := q.Preload("JobType").
		Preload("MechanicProfile").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&lines).Error
	return lines, total, err
}

func (r *InstallationRepository) ListQuotesForUser(userID uuid.UUID, page, limit int) ([]models.InstallationQuote, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := r.db.Model(&models.InstallationQuote{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var quotes []models.InstallationQuote
	err := q.Preload("Lines.MechanicProfile").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&quotes).Error
	return quotes, total, err
}

func (r *InstallationRepository) ExpireQuotesBefore(t time.Time) error {
	return r.db.Model(&models.InstallationQuote{}).
		Where("status = ? AND expires_at < ?", models.QuoteStatusReady, t).
		Update("status", models.QuoteStatusExpired).Error
}

func (r *InstallationRepository) CreateBooking(booking *models.InstallationBooking) error {
	return r.db.Create(booking).Error
}

func (r *InstallationRepository) CreateBookingPayment(p *models.BookingPayment) error {
	return r.db.Create(p).Error
}

func (r *InstallationRepository) GetBookingByID(id uuid.UUID) (*models.InstallationBooking, error) {
	var booking models.InstallationBooking
	err := r.db.Preload("MechanicProfile").
		Preload("QuoteLine.JobType").
		Preload("Payment").
		First(&booking, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *InstallationRepository) GetBookingByIDForUser(id, userID uuid.UUID) (*models.InstallationBooking, error) {
	var booking models.InstallationBooking
	err := r.db.Preload("MechanicProfile").
		Preload("QuoteLine.JobType").
		Preload("Payment").
		First(&booking, "id = ? AND user_id = ?", id, userID).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *InstallationRepository) GetBookingByIDForMechanic(id, mechanicUserID uuid.UUID) (*models.InstallationBooking, error) {
	var booking models.InstallationBooking
	err := r.db.Preload("MechanicProfile").
		Preload("QuoteLine.JobType").
		Preload("Payment").
		First(&booking, "id = ? AND mechanic_user_id = ?", id, mechanicUserID).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *InstallationRepository) UpdateBooking(booking *models.InstallationBooking) error {
	return r.db.Save(booking).Error
}

func (r *InstallationRepository) ListBookingsForUser(userID uuid.UUID, page, limit int) ([]models.InstallationBooking, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := r.db.Model(&models.InstallationBooking{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var bookings []models.InstallationBooking
	err := q.Preload("MechanicProfile").
		Order("scheduled_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&bookings).Error
	return bookings, total, err
}

func (r *InstallationRepository) ListBookingsForMechanic(mechanicUserID uuid.UUID, page, limit int) ([]models.InstallationBooking, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := r.db.Model(&models.InstallationBooking{}).Where("mechanic_user_id = ?", mechanicUserID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var bookings []models.InstallationBooking
	err := q.Preload("MechanicProfile").
		Order("scheduled_at ASC").
		Offset(offset).
		Limit(limit).
		Find(&bookings).Error
	return bookings, total, err
}

func (r *InstallationRepository) BookingExistsForQuote(quoteID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&models.InstallationBooking{}).Where("quote_id = ?", quoteID).Count(&count).Error
	return count > 0, err
}
