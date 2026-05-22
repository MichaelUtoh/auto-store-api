package repositories

import (
	"auto-store-api/internal/models"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DiagramListFilter struct {
	Make              string
	Model             string
	Year              *int
	VehicleSystemCode string
	PublishedOnly     bool
}

type DiagramRepository struct {
	db *gorm.DB
}

func NewDiagramRepository(db *gorm.DB) *DiagramRepository {
	return &DiagramRepository{db: db}
}

func (r *DiagramRepository) ListVehicleSystems() ([]models.VehicleSystem, error) {
	var list []models.VehicleSystem
	err := r.db.Order("display_order ASC, name ASC").Find(&list).Error
	return list, err
}

func (r *DiagramRepository) GetVehicleSystemByCode(code string) (*models.VehicleSystem, error) {
	var vs models.VehicleSystem
	err := r.db.Where("code = ?", strings.ToLower(strings.TrimSpace(code))).First(&vs).Error
	if err != nil {
		return nil, err
	}
	return &vs, nil
}

func (r *DiagramRepository) CreateDiagram(d *models.Diagram) error {
	return r.db.Create(d).Error
}

func (r *DiagramRepository) UpdateDiagram(d *models.Diagram) error {
	return r.db.Save(d).Error
}

func (r *DiagramRepository) DeleteDiagram(id uuid.UUID) error {
	return r.db.Delete(&models.Diagram{}, "id = ?", id).Error
}

func (r *DiagramRepository) GetDiagramByID(id uuid.UUID, includeHotspots bool) (*models.Diagram, error) {
	var d models.Diagram
	q := r.db.Preload("VehicleSystem")
	if includeHotspots {
		q = q.Preload("Hotspots", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC, label ASC")
		})
	}
	err := q.First(&d, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DiagramRepository) ListDiagrams(filter DiagramListFilter, offset, limit int) ([]models.Diagram, int64, error) {
	var diagrams []models.Diagram
	var total int64

	q := r.db.Model(&models.Diagram{}).Preload("VehicleSystem")
	if filter.PublishedOnly {
		q = q.Where("is_published = ?", true)
	}
	if filter.Make != "" {
		q = q.Where("LOWER(make) = ?", strings.ToLower(filter.Make))
	}
	if filter.Model != "" {
		q = q.Where("LOWER(model) = ?", strings.ToLower(filter.Model))
	}
	if filter.Year != nil {
		q = q.Where("year_start <= ? AND year_end >= ?", *filter.Year, *filter.Year)
	}
	if filter.VehicleSystemCode != "" {
		q = q.Joins("JOIN vehicle_systems ON vehicle_systems.id = diagrams.vehicle_system_id").
			Where("vehicle_systems.code = ?", strings.ToLower(filter.VehicleSystemCode))
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("make ASC, model ASC, year_start DESC").
		Offset(offset).Limit(limit).
		Find(&diagrams).Error
	return diagrams, total, err
}

func (r *DiagramRepository) CreateHotspot(h *models.DiagramHotspot) error {
	return r.db.Create(h).Error
}

func (r *DiagramRepository) UpdateHotspot(h *models.DiagramHotspot) error {
	return r.db.Save(h).Error
}

func (r *DiagramRepository) DeleteHotspot(id uuid.UUID) error {
	return r.db.Where("hotspot_id = ?", id).Delete(&models.HotspotProduct{}).Error
}

func (r *DiagramRepository) GetHotspotByID(id uuid.UUID) (*models.DiagramHotspot, error) {
	var h models.DiagramHotspot
	err := r.db.Preload("Diagram.VehicleSystem").First(&h, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *DiagramRepository) ListHotspotsByDiagramID(diagramID uuid.UUID) ([]models.DiagramHotspot, error) {
	var list []models.DiagramHotspot
	err := r.db.Where("diagram_id = ?", diagramID).
		Order("display_order ASC, label ASC").
		Find(&list).Error
	return list, err
}

func (r *DiagramRepository) LinkHotspotProduct(hotspotID, productID uuid.UUID, matchType string) error {
	hp := models.HotspotProduct{
		HotspotID: hotspotID,
		ProductID: productID,
		MatchType: matchType,
	}
	return r.db.Where("hotspot_id = ? AND product_id = ?", hotspotID, productID).
		Assign(models.HotspotProduct{MatchType: matchType}).
		FirstOrCreate(&hp).Error
}

func (r *DiagramRepository) UnlinkHotspotProduct(hotspotID, productID uuid.UUID) error {
	return r.db.Where("hotspot_id = ? AND product_id = ?", hotspotID, productID).
		Delete(&models.HotspotProduct{}).Error
}

func (r *DiagramRepository) GetHotspotProducts(hotspotID uuid.UUID) ([]models.Product, error) {
	var products []models.Product
	err := r.db.Joins("JOIN hotspot_products ON hotspot_products.product_id = products.id").
		Where("hotspot_products.hotspot_id = ?", hotspotID).
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Find(&products).Error
	return products, err
}

func (r *DiagramRepository) FindProductsByOEMAndVehicle(oem string, make, model string, year int) ([]models.Product, error) {
	if oem == "" {
		return nil, nil
	}
	var products []models.Product
	err := r.db.
		Joins("JOIN vehicle_compatibilities ON vehicle_compatibilities.product_id = products.id").
		Where("LOWER(products.manufacturer_part_number) = ?", strings.ToLower(strings.TrimSpace(oem))).
		Where("LOWER(vehicle_compatibilities.make) = ?", strings.ToLower(make)).
		Where("LOWER(vehicle_compatibilities.model) = ?", strings.ToLower(model)).
		Where("vehicle_compatibilities.year_start <= ? AND vehicle_compatibilities.year_end >= ?", year, year).
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Find(&products).Error
	return products, err
}

func (r *DiagramRepository) FindTaxonomiesByLabels(labels []string) ([]models.PartLabelTaxonomy, error) {
	if len(labels) == 0 {
		return nil, nil
	}
	lower := make([]string, len(labels))
	for i, l := range labels {
		lower[i] = strings.ToLower(strings.TrimSpace(l))
	}
	var tax []models.PartLabelTaxonomy
	err := r.db.Preload("VehicleSystem").
		Where("LOWER(label) IN ?", lower).
		Find(&tax).Error
	return tax, err
}

func (r *DiagramRepository) CreatePartIdentification(p *models.PartIdentification) error {
	return r.db.Create(p).Error
}

func (r *DiagramRepository) MatchHotspotsByLabels(diagramID uuid.UUID, labels []string) ([]models.DiagramHotspot, error) {
	if len(labels) == 0 {
		return nil, nil
	}
	var hotspots []models.DiagramHotspot
	q := r.db.Where("diagram_id = ?", diagramID)
	var conds []string
	var args []interface{}
	for _, label := range labels {
		term := "%" + strings.ToLower(strings.TrimSpace(label)) + "%"
		conds = append(conds, "LOWER(label) LIKE ?")
		args = append(args, term)
	}
	q = q.Where(strings.Join(conds, " OR "), args...)
	err := q.Order("display_order ASC").Find(&hotspots).Error
	return hotspots, err
}

func (r *DiagramRepository) MatchHotspotsByPatterns(diagramID uuid.UUID, patterns []string) ([]models.DiagramHotspot, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	var hotspots []models.DiagramHotspot
	q := r.db.Where("diagram_id = ?", diagramID)
	var conds []string
	var args []interface{}
	for _, p := range patterns {
		if p == "" {
			continue
		}
		term := "%" + strings.ToLower(p) + "%"
		conds = append(conds, "LOWER(label) LIKE ?")
		args = append(args, term)
	}
	if len(conds) == 0 {
		return nil, nil
	}
	q = q.Where(strings.Join(conds, " OR "), args...)
	err := q.Find(&hotspots).Error
	return hotspots, err
}
