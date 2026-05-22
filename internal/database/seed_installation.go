package database

import (
	"auto-store-api/internal/models"

	"gorm.io/gorm"
)

var defaultJobTypes = []models.InstallationJobType{
	{Code: "brake_pad_replacement", Name: "Brake Pad Replacement", Description: "Replace front or rear brake pads", BaseLaborMinutes: 90, BaseLaborPrice: 120},
	{Code: "brake_rotor_replacement", Name: "Brake Rotor Replacement", Description: "Replace brake rotors", BaseLaborMinutes: 120, BaseLaborPrice: 150},
	{Code: "oil_change", Name: "Oil Change", Description: "Engine oil and filter change", BaseLaborMinutes: 45, BaseLaborPrice: 60},
	{Code: "battery_replacement", Name: "Battery Replacement", Description: "Replace vehicle battery", BaseLaborMinutes: 30, BaseLaborPrice: 45},
	{Code: "spark_plug_replacement", Name: "Spark Plug Replacement", Description: "Replace spark plugs", BaseLaborMinutes: 60, BaseLaborPrice: 90},
	{Code: "alternator_replacement", Name: "Alternator Replacement", Description: "Replace alternator", BaseLaborMinutes: 120, BaseLaborPrice: 160},
	{Code: "starter_replacement", Name: "Starter Replacement", Description: "Replace starter motor", BaseLaborMinutes: 120, BaseLaborPrice: 160},
	{Code: "suspension_strut_replacement", Name: "Strut Replacement", Description: "Replace front or rear struts", BaseLaborMinutes: 180, BaseLaborPrice: 220},
}

// SeedInstallationJobTypes ensures canonical installation job types exist.
func SeedInstallationJobTypes(db *gorm.DB) error {
	for _, jt := range defaultJobTypes {
		var existing models.InstallationJobType
		err := db.Where("code = ?", jt.Code).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		row := jt
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
