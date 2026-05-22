package services

import (
	"context"
	"errors"
	"time"

	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrMechanicProfileExists    = errors.New("mechanic profile already exists for this user")
	ErrMechanicProfileNotFound  = errors.New("mechanic profile not found")
	ErrMechanicProfileNotEditable = errors.New("mechanic profile cannot be edited in its current status")
	ErrMechanicDocumentNotFound = errors.New("mechanic document not found")
	ErrInvalidMechanicDocument  = errors.New("invalid mechanic document type")
)

type MechanicApplyInput struct {
	BusinessName    string
	Bio             string
	Phone           string
	Street          string
	City            string
	State           string
	PostalCode      string
	Country         string
	Latitude        *float64
	Longitude       *float64
	ServiceRadiusKm float64
	Documents       []MechanicDocumentInput
}

type MechanicDocumentInput struct {
	DocumentType models.MechanicDocumentType
	URL          string
	FileName     string
}

type MechanicUpdateProfileInput struct {
	BusinessName    *string
	Bio             *string
	Phone           *string
	Street          *string
	City            *string
	State           *string
	PostalCode      *string
	Country         *string
	Latitude        *float64
	Longitude       *float64
	ServiceRadiusKm *float64
}

type MechanicService struct {
	mechanicRepo *repositories.MechanicRepository
	installRepo  *repositories.InstallationRepository
	userRepo     *repositories.UserRepository
	notifier     *Notifier
	log          *zap.Logger
	db           *gorm.DB
}

func NewMechanicService(
	mechanicRepo *repositories.MechanicRepository,
	installRepo *repositories.InstallationRepository,
	userRepo *repositories.UserRepository,
	notifier *Notifier,
	log *zap.Logger,
	db *gorm.DB,
) *MechanicService {
	return &MechanicService{mechanicRepo: mechanicRepo, installRepo: installRepo, userRepo: userRepo, notifier: notifier, log: log, db: db}
}

func (s *MechanicService) Apply(userID uuid.UUID, input MechanicApplyInput) (*models.MechanicProfile, error) {
	exists, err := s.mechanicRepo.ProfileExistsForUser(userID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrMechanicProfileExists
	}

	profile := &models.MechanicProfile{
		UserID:          userID,
		BusinessName:    input.BusinessName,
		Bio:             input.Bio,
		Phone:           input.Phone,
		Street:          input.Street,
		City:            input.City,
		State:           input.State,
		PostalCode:      input.PostalCode,
		Country:         input.Country,
		Latitude:        input.Latitude,
		Longitude:       input.Longitude,
		ServiceRadiusKm: input.ServiceRadiusKm,
		Status:          models.MechanicStatusPending,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(profile).Error; err != nil {
			return err
		}
		for _, docInput := range input.Documents {
			if !isValidDocumentType(docInput.DocumentType) {
				return ErrInvalidMechanicDocument
			}
			doc := &models.MechanicDocument{
				MechanicProfileID: profile.ID,
				DocumentType:      docInput.DocumentType,
				URL:               docInput.URL,
				FileName:          docInput.FileName,
				Status:            models.MechanicDocStatusPending,
			}
			if err := tx.Create(doc).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	loaded, err := s.mechanicRepo.GetProfileByUserID(userID)
	if err != nil {
		return nil, err
	}
	s.emitApplyReceived(loaded)
	return loaded, nil
}

func (s *MechanicService) GetProfileByUserID(userID uuid.UUID) (*models.MechanicProfile, error) {
	profile, err := s.mechanicRepo.GetProfileByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMechanicProfileNotFound
		}
		return nil, err
	}
	return profile, nil
}

func (s *MechanicService) GetPublicProfile(profileID uuid.UUID) (*models.MechanicProfile, error) {
	profile, err := s.mechanicRepo.GetProfileByID(profileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMechanicProfileNotFound
		}
		return nil, err
	}
	if profile.Status != models.MechanicStatusVerified {
		return nil, ErrMechanicProfileNotFound
	}
	return profile, nil
}

func (s *MechanicService) UpdateOwnProfile(userID uuid.UUID, input MechanicUpdateProfileInput) (*models.MechanicProfile, error) {
	profile, err := s.GetProfileByUserID(userID)
	if err != nil {
		return nil, err
	}
	if profile.Status != models.MechanicStatusPending && profile.Status != models.MechanicStatusVerified {
		return nil, ErrMechanicProfileNotEditable
	}

	applyProfileUpdates(profile, input)
	if err := s.mechanicRepo.UpdateProfile(profile); err != nil {
		return nil, err
	}
	return s.mechanicRepo.GetProfileByUserID(userID)
}

func (s *MechanicService) AddDocument(userID uuid.UUID, docType models.MechanicDocumentType, url, fileName string) (*models.MechanicDocument, error) {
	if !isValidDocumentType(docType) {
		return nil, ErrInvalidMechanicDocument
	}
	profile, err := s.GetProfileByUserID(userID)
	if err != nil {
		return nil, err
	}
	if profile.Status == models.MechanicStatusSuspended || profile.Status == models.MechanicStatusRejected {
		return nil, ErrMechanicProfileNotEditable
	}

	doc := &models.MechanicDocument{
		MechanicProfileID: profile.ID,
		DocumentType:      docType,
		URL:               url,
		FileName:          fileName,
		Status:            models.MechanicDocStatusPending,
	}
	if err := s.mechanicRepo.CreateDocument(doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *MechanicService) RemoveDocument(userID, documentID uuid.UUID) error {
	profile, err := s.GetProfileByUserID(userID)
	if err != nil {
		return err
	}
	if profile.Status == models.MechanicStatusSuspended {
		return ErrMechanicProfileNotEditable
	}
	if _, err := s.mechanicRepo.GetDocumentByID(documentID, profile.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMechanicDocumentNotFound
		}
		return err
	}
	return s.mechanicRepo.DeleteDocument(documentID, profile.ID)
}

func (s *MechanicService) ListVerified(page, limit int) ([]models.MechanicProfile, int64, error) {
	return s.mechanicRepo.ListVerified(page, limit)
}

func (s *MechanicService) ListForAdmin(status *models.MechanicProfileStatus, page, limit int) ([]models.MechanicProfile, int64, error) {
	return s.mechanicRepo.ListProfiles(status, page, limit)
}

func (s *MechanicService) Verify(userID uuid.UUID) (*models.MechanicProfile, error) {
	profile, err := s.setAdminStatus(userID, models.MechanicStatusVerified, func(profile *models.MechanicProfile, user *models.User) {
		now := time.Now()
		profile.VerifiedAt = &now
		profile.SuspendedAt = nil
		profile.RejectionReason = ""
		user.Role = models.RoleMechanic
	})
	if err != nil {
		return nil, err
	}
	s.emitVerified(profile)
	s.seedInstallServices(profile)
	return profile, nil
}

func (s *MechanicService) seedInstallServices(profile *models.MechanicProfile) {
	if s.installRepo == nil || profile == nil {
		return
	}
	jobTypes, err := s.installRepo.ListJobTypes(true)
	if err != nil {
		s.log.Warn("failed to list job types for mechanic seed", zap.Error(err))
		return
	}
	for _, jt := range jobTypes {
		if err := s.installRepo.UpsertInstallService(&models.MechanicInstallService{
			MechanicProfileID: profile.ID,
			JobTypeID:         jt.ID,
			LaborPrice:        jt.BaseLaborPrice,
			IsActive:          true,
		}); err != nil {
			s.log.Warn("failed to seed mechanic install service", zap.Error(err), zap.String("job_type", jt.Code))
		}
	}
}

func (s *MechanicService) Suspend(userID uuid.UUID, reason string) (*models.MechanicProfile, error) {
	return s.setAdminStatus(userID, models.MechanicStatusSuspended, func(profile *models.MechanicProfile, _ *models.User) {
		now := time.Now()
		profile.SuspendedAt = &now
		if reason != "" {
			profile.RejectionReason = reason
		}
	})
}

func (s *MechanicService) Reject(userID uuid.UUID, reason string) (*models.MechanicProfile, error) {
	return s.setAdminStatus(userID, models.MechanicStatusRejected, func(profile *models.MechanicProfile, user *models.User) {
		profile.RejectionReason = reason
		profile.VerifiedAt = nil
		if user.Role == models.RoleMechanic {
			user.Role = models.RoleCustomer
		}
	})
}

func (s *MechanicService) setAdminStatus(
	userID uuid.UUID,
	status models.MechanicProfileStatus,
	mutate func(profile *models.MechanicProfile, user *models.User),
) (*models.MechanicProfile, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var profile models.MechanicProfile
		if err := tx.First(&profile, "user_id = ?", userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrMechanicProfileNotFound
			}
			return err
		}
		var user models.User
		if err := tx.First(&user, "id = ?", userID).Error; err != nil {
			return err
		}
		profile.Status = status
		mutate(&profile, &user)
		if err := tx.Save(&profile).Error; err != nil {
			return err
		}
		return tx.Save(&user).Error
	})
	if err != nil {
		return nil, err
	}
	return s.mechanicRepo.GetProfileByUserID(userID)
}

func applyProfileUpdates(profile *models.MechanicProfile, input MechanicUpdateProfileInput) {
	if input.BusinessName != nil {
		profile.BusinessName = *input.BusinessName
	}
	if input.Bio != nil {
		profile.Bio = *input.Bio
	}
	if input.Phone != nil {
		profile.Phone = *input.Phone
	}
	if input.Street != nil {
		profile.Street = *input.Street
	}
	if input.City != nil {
		profile.City = *input.City
	}
	if input.State != nil {
		profile.State = *input.State
	}
	if input.PostalCode != nil {
		profile.PostalCode = *input.PostalCode
	}
	if input.Country != nil {
		profile.Country = *input.Country
	}
	if input.Latitude != nil {
		profile.Latitude = input.Latitude
	}
	if input.Longitude != nil {
		profile.Longitude = input.Longitude
	}
	if input.ServiceRadiusKm != nil && *input.ServiceRadiusKm > 0 {
		profile.ServiceRadiusKm = *input.ServiceRadiusKm
	}
}

func isValidDocumentType(t models.MechanicDocumentType) bool {
	switch t {
	case models.MechanicDocLicense, models.MechanicDocInsurance, models.MechanicDocCertification, models.MechanicDocOther:
		return true
	default:
		return false
	}
}

func (s *MechanicService) emitApplyReceived(profile *models.MechanicProfile) {
	if s.notifier == nil || profile == nil {
		return
	}
	if err := s.notifier.MechanicApplyReceived(context.Background(), profile.UserID, profile.BusinessName); err != nil {
		s.log.Warn("failed to send apply received notification", zap.Error(err))
	}
}

func (s *MechanicService) emitVerified(profile *models.MechanicProfile) {
	if s.notifier == nil || profile == nil {
		return
	}
	if err := s.notifier.MechanicVerified(context.Background(), profile.UserID, profile.BusinessName); err != nil {
		s.log.Warn("failed to send mechanic verified notification", zap.Error(err))
	}
}
