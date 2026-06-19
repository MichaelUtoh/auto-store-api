package services

import (
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"

	"github.com/google/uuid"
)

type VehicleCompatibilityService struct {
	repo *repositories.VehicleCompatibilityRepository
}

func NewVehicleCompatibilityService(repo *repositories.VehicleCompatibilityRepository) *VehicleCompatibilityService {
	return &VehicleCompatibilityService{repo: repo}
}

func (s *VehicleCompatibilityService) List(filter repositories.ListVehicleCompatibilityFilter) ([]models.VehicleCompatibility, int64, error) {
	return s.repo.List(filter)
}

func (s *VehicleCompatibilityService) Create(input CreateVehicleCompatibilityInput) (*models.VehicleCompatibility, error) {
	v := models.VehicleCompatibility{
		Make:          input.Make,
		Model:         input.Model,
		Generation:    input.Generation,
		YearStart:     input.YearStart,
		YearEnd:       input.YearEnd,
		Engine:        input.Engine,
		Trim:          input.Trim,
		MarketVariant: input.MarketVariant,
		Notes:         input.Notes,
	}
	if err := s.repo.Create(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *VehicleCompatibilityService) GetByID(id uuid.UUID) (*models.VehicleCompatibility, error) {
	return s.repo.GetByID(id)
}

type CreateVehicleCompatibilityInput struct {
	Make          string
	Model         string
	Generation    string
	YearStart     int
	YearEnd       int
	Engine        string
	Trim          string
	MarketVariant string
	Notes         string
}
