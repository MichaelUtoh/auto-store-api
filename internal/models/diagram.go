package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VehicleSystem groups exploded diagrams (brakes, suspension, engine, …).
type VehicleSystem struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Code         string    `gorm:"uniqueIndex;not null;size:50" json:"code"`
	Name         string    `gorm:"not null" json:"name"`
	Description  string    `gorm:"type:text" json:"description,omitempty"`
	DisplayOrder int       `gorm:"column:display_order;default:0" json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (VehicleSystem) TableName() string { return "vehicle_systems" }

func (v *VehicleSystem) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

// Diagram is an exploded-view image for a vehicle + system.
type Diagram struct {
	ID              uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	VehicleSystemID uuid.UUID      `gorm:"type:uuid;not null;index" json:"vehicle_system_id"`
	Title           string         `gorm:"not null" json:"title"`
	Make            string         `gorm:"not null;index" json:"make"`
	Model           string         `gorm:"not null;index" json:"model"`
	YearStart       int            `gorm:"column:year_start;not null" json:"year_start"`
	YearEnd         int            `gorm:"column:year_end;not null" json:"year_end"`
	ImageURL        string         `gorm:"column:image_url;not null" json:"image_url"`
	SVGOverlayURL   string         `gorm:"column:svg_overlay_url" json:"svg_overlay_url,omitempty"`
	ImageWidth      int            `gorm:"column:image_width;default:0" json:"image_width"`
	ImageHeight     int            `gorm:"column:image_height;default:0" json:"image_height"`
	IsPublished     bool           `gorm:"column:is_published;default:true;index" json:"is_published"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	VehicleSystem VehicleSystem    `gorm:"foreignKey:VehicleSystemID" json:"vehicle_system,omitempty"`
	Hotspots      []DiagramHotspot `gorm:"foreignKey:DiagramID" json:"hotspots,omitempty"`
}

func (Diagram) TableName() string { return "diagrams" }

func (d *Diagram) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

// DiagramHotspot is a clickable region on a diagram (coordinates as % of image size).
type DiagramHotspot struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	DiagramID       uuid.UUID `gorm:"type:uuid;not null;index" json:"diagram_id"`
	Label           string    `gorm:"not null" json:"label"`
	OEMPartNumber   string    `gorm:"column:oem_part_number;size:100" json:"oem_part_number,omitempty"`
	X               float64   `gorm:"not null" json:"x"`
	Y               float64   `gorm:"not null" json:"y"`
	Width           float64   `gorm:"not null" json:"width"`
	Height          float64   `gorm:"not null" json:"height"`
	DisplayOrder    int       `gorm:"column:display_order;default:0" json:"display_order"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Diagram  Diagram   `gorm:"foreignKey:DiagramID" json:"-"`
	Products []Product `gorm:"many2many:hotspot_products;" json:"products,omitempty"`
}

func (DiagramHotspot) TableName() string { return "diagram_hotspots" }

func (h *DiagramHotspot) BeforeCreate(tx *gorm.DB) error {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return nil
}

// HotspotProduct links catalog SKUs to diagram hotspots.
type HotspotProduct struct {
	HotspotID uuid.UUID `gorm:"type:uuid;primaryKey" json:"hotspot_id"`
	ProductID uuid.UUID `gorm:"type:uuid;primaryKey" json:"product_id"`
	MatchType string    `gorm:"column:match_type;size:20;default:'primary'" json:"match_type"`
	CreatedAt time.Time `json:"created_at"`
}

func (HotspotProduct) TableName() string { return "hotspot_products" }

// PartLabelTaxonomy maps CV/AR labels to vehicle systems for part identification.
type PartLabelTaxonomy struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	Label               string     `gorm:"uniqueIndex;not null;size:120" json:"label"`
	VehicleSystemID     *uuid.UUID `gorm:"type:uuid;index" json:"vehicle_system_id,omitempty"`
	HotspotLabelPattern string     `gorm:"column:hotspot_label_pattern;size:120" json:"hotspot_label_pattern,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`

	VehicleSystem *VehicleSystem `gorm:"foreignKey:VehicleSystemID" json:"-"`
}

func (PartLabelTaxonomy) TableName() string { return "part_label_taxonomies" }

func (t *PartLabelTaxonomy) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// PartIdentification stores an AR/CV identification request for audit and debugging.
type PartIdentification struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	UserID          *uuid.UUID `gorm:"type:uuid;index" json:"user_id,omitempty"`
	ImageURL        string    `gorm:"column:image_url;not null" json:"image_url"`
	Make            string    `gorm:"not null;size:100" json:"make"`
	Model           string    `gorm:"not null;size:100" json:"model"`
	Year            int       `gorm:"not null" json:"year"`
	SystemHint      string    `gorm:"column:system_hint;size:50" json:"system_hint,omitempty"`
	LabelsJSON      string    `gorm:"column:labels_json;type:text" json:"-"`
	CandidateCount  int       `gorm:"column:candidate_count;default:0" json:"candidate_count"`
	CreatedAt       time.Time `json:"created_at"`
}

func (PartIdentification) TableName() string { return "part_identifications" }

func (p *PartIdentification) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
