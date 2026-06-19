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
			LowStockThreshold:  in.LowStockThreshold,
			VendorID:           in.VendorID,
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
	LowStockThreshold  int
	VendorID           *uuid.UUID
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

// DeleteProductImage removes one image from a product (by image row UUID).
func (s *ProductService) DeleteProductImage(productID, imageID uuid.UUID) error {
	return s.productRepo.DeleteProductImageByID(productID, imageID)
}

// ReplaceProductImages replaces all images for a product (empty slice removes all images).
func (s *ProductService) ReplaceProductImages(productID uuid.UUID, images []AddImagesInput) error {
	if _, err := s.productRepo.GetByID(productID); err != nil {
		return err
	}
	rows := make([]models.ProductImage, 0, len(images))
	for _, in := range images {
		rows = append(rows, models.ProductImage{
			URL:          in.URL,
			AltText:      in.AltText,
			DisplayOrder: in.DisplayOrder,
			IsPrimary:    in.IsPrimary,
		})
	}
	return s.productRepo.ReplaceProductImages(productID, rows)
}

func (s *ProductService) Delete(id uuid.UUID) error {
	return s.productRepo.Delete(id)
}

func (s *ProductService) List(page, limit int, categorySlug, search, sort string, minPrice, maxPrice *float64) ([]models.Product, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit
	return s.productRepo.List(offset, limit, categorySlug, search, sort, minPrice, maxPrice)
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

// AddCompatibilitiesInput creates a new catalog entry and links it to the product.
type AddCompatibilitiesInput struct {
	Make          string
	Model         string
	Generation    string
	YearStart     int
	YearEnd       int
	Engine        string
	Trim          string
	MarketVariant string
	Notes         string
	LinkNotes     string
}

// LinkCompatibilitiesInput links existing catalog entries to a product.
type LinkCompatibilitiesInput struct {
	CompatibilityID uuid.UUID
	LinkNotes       string
}

func (s *ProductService) ListLinkedCompatibilities(productID uuid.UUID) ([]repositories.LinkedCompatibility, error) {
	if _, err := s.productRepo.GetByID(productID); err != nil {
		return nil, err
	}
	return s.compatRepo.GetLinkedByProductID(productID)
}

// LinkCompatibilities attaches existing catalog compatibilities to a product.
func (s *ProductService) LinkCompatibilities(productID uuid.UUID, items []LinkCompatibilitiesInput) ([]repositories.LinkedCompatibility, error) {
	if _, err := s.productRepo.GetByID(productID); err != nil {
		return nil, err
	}
	for _, in := range items {
		if _, err := s.compatRepo.GetByID(in.CompatibilityID); err != nil {
			return nil, err
		}
		if err := s.compatRepo.LinkProduct(productID, in.CompatibilityID, in.LinkNotes); err != nil {
			return nil, err
		}
	}
	return s.compatRepo.GetLinkedByProductID(productID)
}

// AddCompatibilities creates catalog entries (deduped) and links them to a product.
func (s *ProductService) AddCompatibilities(productID uuid.UUID, items []AddCompatibilitiesInput) ([]repositories.LinkedCompatibility, error) {
	if _, err := s.productRepo.GetByID(productID); err != nil {
		return nil, err
	}
	for _, in := range items {
		v, err := s.compatRepo.FindOrCreate(models.VehicleCompatibility{
			Make:          in.Make,
			Model:         in.Model,
			Generation:    in.Generation,
			YearStart:     in.YearStart,
			YearEnd:       in.YearEnd,
			Engine:        in.Engine,
			Trim:          in.Trim,
			MarketVariant: in.MarketVariant,
			Notes:         in.Notes,
		})
		if err != nil {
			return nil, err
		}
		linkNotes := in.LinkNotes
		if linkNotes == "" {
			linkNotes = in.Notes
		}
		if err := s.compatRepo.LinkProduct(productID, v.ID, linkNotes); err != nil {
			return nil, err
		}
	}
	return s.compatRepo.GetLinkedByProductID(productID)
}
