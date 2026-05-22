package database

import (
	"auto-store-api/internal/models"

	"gorm.io/gorm"
)

var defaultVehicleSystems = []models.VehicleSystem{
	{Code: "brakes", Name: "Brakes", Description: "Brake pads, rotors, calipers, lines", DisplayOrder: 1},
	{Code: "suspension", Name: "Suspension", Description: "Struts, shocks, control arms, bushings", DisplayOrder: 2},
	{Code: "engine", Name: "Engine", Description: "Engine bay components and accessories", DisplayOrder: 3},
	{Code: "electrical", Name: "Electrical", Description: "Battery, alternator, starter, wiring", DisplayOrder: 4},
	{Code: "cooling", Name: "Cooling", Description: "Radiator, hoses, water pump, thermostat", DisplayOrder: 5},
}

var defaultPartLabelTaxonomies = []models.PartLabelTaxonomy{
	{Label: "brake pad", HotspotLabelPattern: "pad"},
	{Label: "brake rotor", HotspotLabelPattern: "rotor"},
	{Label: "brake caliper", HotspotLabelPattern: "caliper"},
	{Label: "strut", HotspotLabelPattern: "strut"},
	{Label: "shock absorber", HotspotLabelPattern: "shock"},
	{Label: "alternator", HotspotLabelPattern: "alternator"},
	{Label: "starter motor", HotspotLabelPattern: "starter"},
	{Label: "battery", HotspotLabelPattern: "battery"},
}

// SeedVehicleSystems ensures canonical vehicle systems exist for the part finder.
func SeedVehicleSystems(db *gorm.DB) error {
	for _, vs := range defaultVehicleSystems {
		var existing models.VehicleSystem
		err := db.Where("code = ?", vs.Code).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		row := vs
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

// SeedPartLabelTaxonomies seeds CV label → hotspot matching hints.
func SeedPartLabelTaxonomies(db *gorm.DB) error {
	for _, t := range defaultPartLabelTaxonomies {
		var existing models.PartLabelTaxonomy
		err := db.Where("label = ?", t.Label).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		row := t
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
