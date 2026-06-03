package database

import (
	"auto-store-api/internal/models"
	"strings"

	"gorm.io/gorm"
)

type sampleHotspot struct {
	Label         string
	OEMPartNumber string
	X, Y, W, H    float64
	DisplayOrder  int
	SKUs          []string // optional catalog links when products exist
}

// SeedSampleDiagrams adds a demo Toyota Camry front-brakes diagram with hotspots.
// Idempotent: skips if a published Camry brakes diagram already exists.
func SeedSampleDiagrams(db *gorm.DB) error {
	var brakes models.VehicleSystem
	if err := db.Where("code = ?", "brakes").First(&brakes).Error; err != nil {
		return err
	}

	var existing int64
	if err := db.Model(&models.Diagram{}).
		Where("LOWER(make) = ? AND LOWER(model) = ? AND vehicle_system_id = ?",
			"toyota", "camry", brakes.ID).
		Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	diagram := models.Diagram{
		VehicleSystemID: brakes.ID,
		Title:           "2018 Toyota Camry — Front Brake Assembly",
		Make:            "Toyota",
		Model:           "Camry",
		YearStart:       2015,
		YearEnd:         2020,
		ImageURL:        "https://placehold.co/1200x800/0f172a/94a3b8/png?text=2018+Toyota+Camry+Front+Brakes",
		ImageWidth:      1200,
		ImageHeight:     800,
		IsPublished:     true,
	}
	if err := db.Create(&diagram).Error; err != nil {
		return err
	}

	hotspots := []sampleHotspot{
		{Label: "Front Brake Pad (Left)", OEMPartNumber: "ST-1234", X: 14, Y: 54, W: 14, H: 11, DisplayOrder: 1, SKUs: []string{"BP-CAMRY-F"}},
		{Label: "Front Brake Pad (Right)", OEMPartNumber: "ST-1234", X: 72, Y: 54, W: 14, H: 11, DisplayOrder: 2, SKUs: []string{"BP-CAMRY-F"}},
		{Label: "Front Brake Rotor (Left)", OEMPartNumber: "RT-4455", X: 8, Y: 62, W: 20, H: 14, DisplayOrder: 3},
		{Label: "Front Brake Rotor (Right)", OEMPartNumber: "RT-4455", X: 70, Y: 62, W: 20, H: 14, DisplayOrder: 4},
		{Label: "Brake Caliper (Left)", OEMPartNumber: "CL-7788", X: 22, Y: 46, W: 15, H: 13, DisplayOrder: 5},
		{Label: "Brake Caliper (Right)", OEMPartNumber: "CL-7788", X: 63, Y: 46, W: 15, H: 13, DisplayOrder: 6},
		{Label: "Brake Hose (Left)", OEMPartNumber: "BH-2210", X: 26, Y: 32, W: 9, H: 22, DisplayOrder: 7},
		{Label: "Brake Hose (Right)", OEMPartNumber: "BH-2210", X: 65, Y: 32, W: 9, H: 22, DisplayOrder: 8},
		{Label: "Brake Fluid Reservoir", OEMPartNumber: "BF-9901", X: 40, Y: 14, W: 18, H: 12, DisplayOrder: 9},
	}

	for _, sh := range hotspots {
		h := models.DiagramHotspot{
			DiagramID:     diagram.ID,
			Label:         sh.Label,
			OEMPartNumber: sh.OEMPartNumber,
			X:             sh.X,
			Y:             sh.Y,
			Width:         sh.W,
			Height:        sh.H,
			DisplayOrder:  sh.DisplayOrder,
		}
		if err := db.Create(&h).Error; err != nil {
			return err
		}
		if err := linkSampleHotspotProducts(db, &h, sh, diagram); err != nil {
			return err
		}
	}
	return nil
}

func linkSampleHotspotProducts(db *gorm.DB, hotspot *models.DiagramHotspot, sh sampleHotspot, diagram models.Diagram) error {
	linked := make(map[string]bool)

	for _, sku := range sh.SKUs {
		var product models.Product
		if err := db.Where("sku = ?", sku).First(&product).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}
		if linked[product.ID.String()] {
			continue
		}
		if err := db.Create(&models.HotspotProduct{
			HotspotID: hotspot.ID,
			ProductID: product.ID,
			MatchType: "primary",
		}).Error; err != nil {
			return err
		}
		linked[product.ID.String()] = true
	}

	if sh.OEMPartNumber != "" {
		var products []models.Product
		err := db.
			Joins("JOIN vehicle_compatibilities ON vehicle_compatibilities.product_id = products.id").
			Where("LOWER(products.manufacturer_part_number) = ?", strings.ToLower(sh.OEMPartNumber)).
			Where("LOWER(vehicle_compatibilities.make) = ?", "toyota").
			Where("LOWER(vehicle_compatibilities.model) = ?", "camry").
			Where("vehicle_compatibilities.year_start <= ? AND vehicle_compatibilities.year_end >= ?", 2018, 2018).
			Find(&products).Error
		if err != nil {
			return err
		}
		for _, p := range products {
			if linked[p.ID.String()] {
				continue
			}
			if err := db.Create(&models.HotspotProduct{
				HotspotID: hotspot.ID,
				ProductID: p.ID,
				MatchType: "alternate",
			}).Error; err != nil {
				return err
			}
			linked[p.ID.String()] = true
		}
	}
	return nil
}
