package database

import (
	"auto-store-api/internal/models"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const legacyVehicleCompatTable = "vehicle_compatibilities_legacy"

type legacyVehicleCompatibilityRow struct {
	ID        uuid.UUID `gorm:"column:id"`
	ProductID uuid.UUID `gorm:"column:product_id"`
	Make      string    `gorm:"column:make"`
	Model     string    `gorm:"column:model"`
	YearStart int       `gorm:"column:year_start"`
	YearEnd   int       `gorm:"column:year_end"`
	Engine    string    `gorm:"column:engine"`
	Trim      string    `gorm:"column:trim"`
	Notes     string    `gorm:"column:notes"`
}

func (legacyVehicleCompatibilityRow) TableName() string { return legacyVehicleCompatTable }

type intVehicleCompatibilityRow struct {
	ID            int       `gorm:"column:id"`
	Make          string    `gorm:"column:make"`
	Model         string    `gorm:"column:model"`
	Generation    string    `gorm:"column:generation"`
	YearStart     int       `gorm:"column:year_start"`
	YearEnd       int       `gorm:"column:year_end"`
	Engine        string    `gorm:"column:engine"`
	Trim          string    `gorm:"column:trim"`
	MarketVariant string    `gorm:"column:market_variant"`
	Notes         string    `gorm:"column:notes"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

type intProductVehicleCompatibilityRow struct {
	ProductID              uuid.UUID `gorm:"column:product_id"`
	VehicleCompatibilityID int       `gorm:"column:vehicle_compatibility_id"`
	Notes                  string    `gorm:"column:notes"`
}

// prepareLegacyVehicleCompatibilityTable renames product-owned rows before catalog AutoMigrate.
func prepareLegacyVehicleCompatibilityTable(db *gorm.DB) error {
	if !db.Migrator().HasTable("vehicle_compatibilities") {
		return nil
	}
	if !db.Migrator().HasColumn("vehicle_compatibilities", "product_id") {
		return nil
	}
	if db.Migrator().HasTable(legacyVehicleCompatTable) {
		return nil
	}
	return db.Migrator().RenameTable("vehicle_compatibilities", legacyVehicleCompatTable)
}

// migrateVehicleCompatibilityIntToUUID converts catalog + junction from serial IDs to UUIDs.
func migrateVehicleCompatibilityIntToUUID(db *gorm.DB) error {
	if !db.Migrator().HasTable("vehicle_compatibilities") {
		return nil
	}
	var dataType string
	row := db.Raw(`SELECT data_type FROM information_schema.columns
		WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'vehicle_compatibilities' AND column_name = 'id'`).Row()
	if err := row.Scan(&dataType); err != nil {
		return nil
	}
	if dataType == "uuid" {
		return nil
	}
	if dataType != "integer" && dataType != "bigint" && dataType != "smallint" {
		return nil
	}

	var catalog []intVehicleCompatibilityRow
	if err := db.Table("vehicle_compatibilities").Find(&catalog).Error; err != nil {
		return err
	}

	var links []intProductVehicleCompatibilityRow
	if db.Migrator().HasTable("product_vehicle_compatibilities") {
		if err := db.Table("product_vehicle_compatibilities").Find(&links).Error; err != nil {
			return err
		}
	}

	idMap := make(map[int]uuid.UUID, len(catalog))
	newCatalog := make([]models.VehicleCompatibility, len(catalog))
	for i, row := range catalog {
		newID := uuid.New()
		idMap[row.ID] = newID
		market := row.MarketVariant
		if market == "" {
			market = "nigeria"
		}
		newCatalog[i] = models.VehicleCompatibility{
			ID:            newID,
			Make:          row.Make,
			Model:         row.Model,
			Generation:    row.Generation,
			YearStart:     row.YearStart,
			YearEnd:       row.YearEnd,
			Engine:        row.Engine,
			Trim:          row.Trim,
			MarketVariant: market,
			Notes:         row.Notes,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasTable("product_vehicle_compatibilities") {
			if err := tx.Migrator().DropTable("product_vehicle_compatibilities"); err != nil {
				return err
			}
		}
		if err := tx.Migrator().DropTable("vehicle_compatibilities"); err != nil {
			return err
		}
		if err := tx.AutoMigrate(&models.VehicleCompatibility{}, &models.ProductVehicleCompatibility{}); err != nil {
			return err
		}
		if len(newCatalog) > 0 {
			if err := tx.Create(&newCatalog).Error; err != nil {
				return err
			}
		}
		for _, link := range links {
			compatID, ok := idMap[link.VehicleCompatibilityID]
			if !ok {
				continue
			}
			newLink := models.ProductVehicleCompatibility{
				ProductID:              link.ProductID,
				VehicleCompatibilityID: compatID,
				Notes:                  link.Notes,
			}
			if err := tx.Where("product_id = ? AND vehicle_compatibility_id = ?", link.ProductID, compatID).
				FirstOrCreate(&newLink).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// MigrateVehicleCompatibilityM2M backfills catalog + junction from legacy product-owned rows.
func MigrateVehicleCompatibilityM2M(db *gorm.DB) error {
	if !db.Migrator().HasTable(legacyVehicleCompatTable) {
		return nil
	}
	if !db.Migrator().HasTable("vehicle_compatibilities") || !db.Migrator().HasTable("product_vehicle_compatibilities") {
		return nil
	}

	var legacy []legacyVehicleCompatibilityRow
	if err := db.Table(legacyVehicleCompatTable).Find(&legacy).Error; err != nil {
		return err
	}

	for _, row := range legacy {
		v := models.VehicleCompatibility{
			Make:          row.Make,
			Model:         row.Model,
			YearStart:     row.YearStart,
			YearEnd:       row.YearEnd,
			Engine:        row.Engine,
			Trim:          row.Trim,
			MarketVariant: "nigeria",
			Notes:         row.Notes,
		}
		var existing models.VehicleCompatibility
		err := db.Where(
			`LOWER(make) = ? AND LOWER(model) = ? AND generation = '' AND year_start = ? AND year_end = ?
			 AND LOWER(engine) = ? AND trim = ? AND LOWER(market_variant) = 'nigeria'`,
			strings.ToLower(row.Make),
			strings.ToLower(row.Model),
			row.YearStart,
			row.YearEnd,
			strings.ToLower(row.Engine),
			row.Trim,
		).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := db.Create(&v).Error; err != nil {
				return err
			}
			existing = v
		} else if err != nil {
			return err
		}

		link := models.ProductVehicleCompatibility{
			ProductID:              row.ProductID,
			VehicleCompatibilityID: existing.ID,
			Notes:                  row.Notes,
		}
		if err := db.Where("product_id = ? AND vehicle_compatibility_id = ?", row.ProductID, existing.ID).
			FirstOrCreate(&link).Error; err != nil {
			return err
		}
	}

	return db.Migrator().DropTable(legacyVehicleCompatTable)
}
