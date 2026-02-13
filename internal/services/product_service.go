package services

import (
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductService struct {
	productRepo  *repositories.ProductRepository
	categoryRepo *repositories.CategoryRepository
	compatRepo   *repositories.VehicleCompatibilityRepository
	db           *gorm.DB
}

func NewProductService(
	productRepo *repositories.ProductRepository,
	categoryRepo *repositories.CategoryRepository,
	compatRepo *repositories.VehicleCompatibilityRepository,
	db *gorm.DB,
) *ProductService {
	return &ProductService{
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
		compatRepo:   compatRepo,
		db:           db,
	}
}

func (s *ProductService) Create(p *models.Product, categoryIDs, tagIDs []uuid.UUID) error {
	if err := s.productRepo.Create(p); err != nil {
		return err
	}
	if len(categoryIDs) > 0 {
		_ = s.productRepo.SetCategories(p.ID, categoryIDs)
	}
	if len(tagIDs) > 0 {
		_ = s.productRepo.SetTags(p.ID, tagIDs)
	}
	return nil
}

// BatchProductFailure records a failed product creation in a batch.
type BatchProductFailure struct {
	Index   int
	SKU     string
	Message string
}

const MaxBatchSize = 100

// CreateBatch creates multiple products. Returns created products (with original index) and any failures (best-effort).
func (s *ProductService) CreateBatch(items []CreateProductInput) (created []BatchProductSuccess, failed []BatchProductFailure) {
	for i, in := range items {
		p := &models.Product{
			SKU:                in.SKU,
			Name:               in.Name,
			Description:        in.Description,
			Brand:              in.Brand,
			ManufacturerPartNo: in.ManufacturerPartNo,
			Price:              in.Price,
			CostPrice:          in.CostPrice,
			StockQuantity:      in.StockQuantity,
			Weight:             in.Weight,
			Dimensions:         in.Dimensions,
			Condition:          in.Condition,
			WarrantyMonths:     in.WarrantyMonths,
		}
		if err := s.Create(p, in.CategoryIDs, in.TagIDs); err != nil {
			failed = append(failed, BatchProductFailure{Index: i, SKU: in.SKU, Message: err.Error()})
			continue
		}
		full, _ := s.GetByID(p.ID)
		prod := p
		if full != nil {
			prod = full
		}
		created = append(created, BatchProductSuccess{Index: i, Product: prod})
	}
	return created, failed
}

// BatchProductSuccess records a successful product creation in a batch.
type BatchProductSuccess struct {
	Index   int
	Product *models.Product
}

// CreateProductInput is input for a single product in a batch.
type CreateProductInput struct {
	SKU                string
	Name               string
	Description        string
	Brand              string
	ManufacturerPartNo string
	Price              float64
	CostPrice          float64
	StockQuantity      int
	Weight             float64
	Dimensions         string
	Condition          models.ProductCondition
	WarrantyMonths     int
	CategoryIDs        []uuid.UUID
	TagIDs             []uuid.UUID
}

func (s *ProductService) GetByID(id uuid.UUID) (*models.Product, error) {
	return s.productRepo.GetByID(id, "Categories", "Tags", "Images", "Specifications", "Compatibilities")
}

func (s *ProductService) Update(p *models.Product, categoryIDs, tagIDs []uuid.UUID) error {
	if err := s.productRepo.Update(p); err != nil {
		return err
	}
	if categoryIDs != nil {
		_ = s.productRepo.SetCategories(p.ID, categoryIDs)
	}
	if tagIDs != nil {
		_ = s.productRepo.SetTags(p.ID, tagIDs)
	}
	return nil
}

func (s *ProductService) Delete(id uuid.UUID) error {
	return s.productRepo.Delete(id)
}

func (s *ProductService) List(page, limit int, filters map[string]interface{}) ([]models.Product, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.productRepo.List(offset, limit, filters)
}

func (s *ProductService) Search(params repositories.SearchParams) (*repositories.SearchResult, error) {
	return s.productRepo.Search(params)
}

func (s *ProductService) ListByCategoryIDs(categoryIDs []uuid.UUID, page, limit int) ([]models.Product, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.productRepo.ListByIDs(categoryIDs, offset, limit)
}

// AddImagesInput is a single image to add.
type AddImagesInput struct {
	URL          string
	AltText      string
	DisplayOrder int
	IsPrimary    bool
}

// AddImages adds images to a product by ID. If any image has IsPrimary true, other images for the product are unset as primary.
func (s *ProductService) AddImages(productID uuid.UUID, images []AddImagesInput) ([]models.ProductImage, error) {
	if _, err := s.productRepo.GetByID(productID); err != nil {
		return nil, err
	}
	hasPrimary := false
	for _, img := range images {
		if img.IsPrimary {
			hasPrimary = true
			break
		}
	}
	if hasPrimary {
		_ = s.productRepo.UnsetPrimaryImages(productID)
	}
	var created []models.ProductImage
	for _, in := range images {
		img := models.ProductImage{
			ProductID:    productID,
			URL:          in.URL,
			AltText:      in.AltText,
			DisplayOrder: in.DisplayOrder,
			IsPrimary:    in.IsPrimary,
		}
		if err := s.productRepo.CreateProductImage(&img); err != nil {
			return nil, err
		}
		created = append(created, img)
	}
	return created, nil
}

// AddCompatibilitiesInput is a single vehicle compatibility to add.
type AddCompatibilitiesInput struct {
	Make      string
	Model     string
	YearStart int
	YearEnd   int
	Engine    string
	Trim      string
	Notes     string
}

// AddCompatibilities adds vehicle compatibilities to a product by ID.
func (s *ProductService) AddCompatibilities(productID uuid.UUID, items []AddCompatibilitiesInput) ([]models.VehicleCompatibility, error) {
	if _, err := s.productRepo.GetByID(productID); err != nil {
		return nil, err
	}
	var created []models.VehicleCompatibility
	for _, in := range items {
		v := models.VehicleCompatibility{
			ProductID: productID,
			Make:      in.Make,
			Model:     in.Model,
			YearStart: in.YearStart,
			YearEnd:   in.YearEnd,
			Engine:    in.Engine,
			Trim:      in.Trim,
			Notes:     in.Notes,
		}
		if err := s.compatRepo.Create(&v); err != nil {
			return nil, err
		}
		created = append(created, v)
	}
	return created, nil
}
