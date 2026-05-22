package services

import (
	"errors"
	"strings"

	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrDiagramNotFound      = errors.New("diagram not found")
	ErrHotspotNotFound      = errors.New("hotspot not found")
	ErrVehicleSystemNotFound = errors.New("vehicle system not found")
	ErrInvalidDiagramYears  = errors.New("year_end must be >= year_start")
)

type DiagramListParams struct {
	Make              string
	Model             string
	Year              *int
	VehicleSystemCode string
	Page              int
	Limit             int
}

type CreateDiagramInput struct {
	VehicleSystemCode string
	Title             string
	Make              string
	Model             string
	YearStart         int
	YearEnd           int
	ImageURL          string
	SVGOverlayURL     string
	ImageWidth        int
	ImageHeight       int
	IsPublished       bool
}

type UpdateDiagramInput struct {
	Title         *string
	ImageURL      *string
	SVGOverlayURL *string
	ImageWidth    *int
	ImageHeight   *int
	IsPublished   *bool
	YearStart     *int
	YearEnd       *int
}

type CreateHotspotInput struct {
	Label         string
	OEMPartNumber string
	X             float64
	Y             float64
	Width         float64
	Height        float64
	DisplayOrder  int
}

type UpdateHotspotInput struct {
	Label         *string
	OEMPartNumber *string
	X             *float64
	Y             *float64
	Width         *float64
	Height        *float64
	DisplayOrder  *int
}

type DiagramService struct {
	diagramRepo *repositories.DiagramRepository
	productRepo *repositories.ProductRepository
	db          *gorm.DB
}

func NewDiagramService(
	diagramRepo *repositories.DiagramRepository,
	productRepo *repositories.ProductRepository,
	db *gorm.DB,
) *DiagramService {
	return &DiagramService{
		diagramRepo: diagramRepo,
		productRepo: productRepo,
		db:          db,
	}
}

func (s *DiagramService) ListVehicleSystems() ([]models.VehicleSystem, error) {
	return s.diagramRepo.ListVehicleSystems()
}

func (s *DiagramService) List(params DiagramListParams) ([]models.Diagram, int64, error) {
	page, limit := normalizePage(params.Page, params.Limit)
	offset := (page - 1) * limit
	return s.diagramRepo.ListDiagrams(repositories.DiagramListFilter{
		Make:              params.Make,
		Model:             params.Model,
		Year:              params.Year,
		VehicleSystemCode: params.VehicleSystemCode,
		PublishedOnly:     true,
	}, offset, limit)
}

func (s *DiagramService) GetByID(id uuid.UUID, includeHotspots bool) (*models.Diagram, error) {
	d, err := s.diagramRepo.GetDiagramByID(id, includeHotspots)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDiagramNotFound
		}
		return nil, err
	}
	if !d.IsPublished {
		return nil, ErrDiagramNotFound
	}
	return d, nil
}

func (s *DiagramService) ListHotspots(diagramID uuid.UUID) ([]models.DiagramHotspot, error) {
	if _, err := s.GetByID(diagramID, false); err != nil {
		return nil, err
	}
	return s.diagramRepo.ListHotspotsByDiagramID(diagramID)
}

func (s *DiagramService) GetHotspotProducts(diagramID, hotspotID uuid.UUID, year *int) ([]models.Product, error) {
	d, err := s.GetByID(diagramID, false)
	if err != nil {
		return nil, err
	}
	h, err := s.diagramRepo.GetHotspotByID(hotspotID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrHotspotNotFound
		}
		return nil, err
	}
	if h.DiagramID != diagramID {
		return nil, ErrHotspotNotFound
	}

	linked, err := s.diagramRepo.GetHotspotProducts(hotspotID)
	if err != nil {
		return nil, err
	}

	matchYear := d.YearEnd
	if year != nil {
		matchYear = *year
	}
	oemMatches, err := s.diagramRepo.FindProductsByOEMAndVehicle(
		h.OEMPartNumber, d.Make, d.Model, matchYear,
	)
	if err != nil {
		return nil, err
	}

	return mergeProducts(linked, oemMatches), nil
}

func (s *DiagramService) Create(input CreateDiagramInput) (*models.Diagram, error) {
	if input.YearEnd < input.YearStart {
		return nil, ErrInvalidDiagramYears
	}
	vs, err := s.diagramRepo.GetVehicleSystemByCode(input.VehicleSystemCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVehicleSystemNotFound
		}
		return nil, err
	}
	d := &models.Diagram{
		VehicleSystemID: vs.ID,
		Title:           strings.TrimSpace(input.Title),
		Make:            strings.TrimSpace(input.Make),
		Model:           strings.TrimSpace(input.Model),
		YearStart:       input.YearStart,
		YearEnd:         input.YearEnd,
		ImageURL:        strings.TrimSpace(input.ImageURL),
		SVGOverlayURL:   strings.TrimSpace(input.SVGOverlayURL),
		ImageWidth:      input.ImageWidth,
		ImageHeight:     input.ImageHeight,
		IsPublished:     input.IsPublished,
	}
	if err := s.diagramRepo.CreateDiagram(d); err != nil {
		return nil, err
	}
	return s.diagramRepo.GetDiagramByID(d.ID, false)
}

func (s *DiagramService) Update(id uuid.UUID, input UpdateDiagramInput) (*models.Diagram, error) {
	d, err := s.diagramRepo.GetDiagramByID(id, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDiagramNotFound
		}
		return nil, err
	}
	if input.Title != nil {
		d.Title = strings.TrimSpace(*input.Title)
	}
	if input.ImageURL != nil {
		d.ImageURL = strings.TrimSpace(*input.ImageURL)
	}
	if input.SVGOverlayURL != nil {
		d.SVGOverlayURL = strings.TrimSpace(*input.SVGOverlayURL)
	}
	if input.ImageWidth != nil {
		d.ImageWidth = *input.ImageWidth
	}
	if input.ImageHeight != nil {
		d.ImageHeight = *input.ImageHeight
	}
	if input.IsPublished != nil {
		d.IsPublished = *input.IsPublished
	}
	if input.YearStart != nil {
		d.YearStart = *input.YearStart
	}
	if input.YearEnd != nil {
		d.YearEnd = *input.YearEnd
	}
	if d.YearEnd < d.YearStart {
		return nil, ErrInvalidDiagramYears
	}
	if err := s.diagramRepo.UpdateDiagram(d); err != nil {
		return nil, err
	}
	return s.diagramRepo.GetDiagramByID(id, false)
}

func (s *DiagramService) Delete(id uuid.UUID) error {
	if _, err := s.diagramRepo.GetDiagramByID(id, false); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDiagramNotFound
		}
		return err
	}
	return s.diagramRepo.DeleteDiagram(id)
}

func (s *DiagramService) CreateHotspot(diagramID uuid.UUID, input CreateHotspotInput) (*models.DiagramHotspot, error) {
	if _, err := s.diagramRepo.GetDiagramByID(diagramID, false); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDiagramNotFound
		}
		return nil, err
	}
	h := &models.DiagramHotspot{
		DiagramID:     diagramID,
		Label:         strings.TrimSpace(input.Label),
		OEMPartNumber: strings.TrimSpace(input.OEMPartNumber),
		X:             input.X,
		Y:             input.Y,
		Width:         input.Width,
		Height:        input.Height,
		DisplayOrder:  input.DisplayOrder,
	}
	if err := s.diagramRepo.CreateHotspot(h); err != nil {
		return nil, err
	}
	return s.diagramRepo.GetHotspotByID(h.ID)
}

func (s *DiagramService) UpdateHotspot(diagramID, hotspotID uuid.UUID, input UpdateHotspotInput) (*models.DiagramHotspot, error) {
	h, err := s.diagramRepo.GetHotspotByID(hotspotID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrHotspotNotFound
		}
		return nil, err
	}
	if h.DiagramID != diagramID {
		return nil, ErrHotspotNotFound
	}
	if input.Label != nil {
		h.Label = strings.TrimSpace(*input.Label)
	}
	if input.OEMPartNumber != nil {
		h.OEMPartNumber = strings.TrimSpace(*input.OEMPartNumber)
	}
	if input.X != nil {
		h.X = *input.X
	}
	if input.Y != nil {
		h.Y = *input.Y
	}
	if input.Width != nil {
		h.Width = *input.Width
	}
	if input.Height != nil {
		h.Height = *input.Height
	}
	if input.DisplayOrder != nil {
		h.DisplayOrder = *input.DisplayOrder
	}
	if err := s.diagramRepo.UpdateHotspot(h); err != nil {
		return nil, err
	}
	return s.diagramRepo.GetHotspotByID(hotspotID)
}

func (s *DiagramService) DeleteHotspot(diagramID, hotspotID uuid.UUID) error {
	h, err := s.diagramRepo.GetHotspotByID(hotspotID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrHotspotNotFound
		}
		return err
	}
	if h.DiagramID != diagramID {
		return ErrHotspotNotFound
	}
	if err := s.diagramRepo.DeleteHotspot(hotspotID); err != nil {
		return err
	}
	return s.db.Delete(&models.DiagramHotspot{}, "id = ?", hotspotID).Error
}

func (s *DiagramService) LinkProduct(diagramID, hotspotID, productID uuid.UUID, matchType string) error {
	if _, err := s.diagramRepo.GetDiagramByID(diagramID, false); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDiagramNotFound
		}
		return err
	}
	h, err := s.diagramRepo.GetHotspotByID(hotspotID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrHotspotNotFound
		}
		return err
	}
	if h.DiagramID != diagramID {
		return ErrHotspotNotFound
	}
	if _, err := s.productRepo.GetByID(productID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProductNotFound
		}
		return err
	}
	if matchType == "" {
		matchType = "primary"
	}
	return s.diagramRepo.LinkHotspotProduct(hotspotID, productID, matchType)
}

func (s *DiagramService) UnlinkProduct(diagramID, hotspotID, productID uuid.UUID) error {
	h, err := s.diagramRepo.GetHotspotByID(hotspotID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrHotspotNotFound
		}
		return err
	}
	if h.DiagramID != diagramID {
		return ErrHotspotNotFound
	}
	return s.diagramRepo.UnlinkHotspotProduct(hotspotID, productID)
}

func mergeProducts(a, b []models.Product) []models.Product {
	seen := make(map[uuid.UUID]bool)
	var out []models.Product
	for _, p := range append(a, b...) {
		if seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		out = append(out, p)
	}
	return out
}

func normalizePage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
