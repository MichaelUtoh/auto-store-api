package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductCondition string

const (
	ConditionNew         ProductCondition = "new"
	ConditionRefurbished ProductCondition = "refurbished"
	ConditionUsed        ProductCondition = "used"
)

type Product struct {
	ID                    uuid.UUID         `gorm:"type:uuid;primary_key" json:"id"`
	SKU                   string            `gorm:"uniqueIndex;not null" json:"sku"`
	Name                  string            `gorm:"not null" json:"name"`
	Description           string            `gorm:"type:text" json:"description"`
	Brand                 string            `json:"brand"`
	ManufacturerPartNo    string            `gorm:"column:manufacturer_part_number" json:"manufacturer_part_number"`
	Price                 float64           `gorm:"type:decimal(12,2);not null" json:"price"`
	CostPrice             float64           `gorm:"column:cost_price;type:decimal(12,2)" json:"cost_price"`
	StockQuantity         int               `gorm:"column:stock_quantity;default:0" json:"stock_quantity"`
	Weight                float64           `gorm:"type:decimal(10,2)" json:"weight"`
	Dimensions            string            `gorm:"type:varchar(100)" json:"dimensions"`
	Condition             ProductCondition  `gorm:"type:varchar(20);default:'new'" json:"condition"`
	WarrantyMonths        int               `gorm:"column:warranty_months" json:"warranty_months"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
	DeletedAt             gorm.DeletedAt    `gorm:"index" json:"-"`

	Categories     []Category     `gorm:"many2many:product_categories;" json:"categories,omitempty"`
	Tags           []Tag          `gorm:"many2many:product_tags;" json:"tags,omitempty"`
	Images         []ProductImage `gorm:"foreignKey:ProductID" json:"images,omitempty"`
	Specifications []Specification `gorm:"foreignKey:ProductID" json:"specifications,omitempty"`
	Compatibilities []VehicleCompatibility `gorm:"foreignKey:ProductID" json:"compatibilities,omitempty"`
	Reviews        []Review       `gorm:"foreignKey:ProductID" json:"reviews,omitempty"`
}

func (Product) TableName() string { return "products" }

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type Category struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	ParentID    *uuid.UUID     `gorm:"type:uuid;column:parent_id" json:"parent_id"`
	Name        string         `gorm:"not null" json:"name"`
	Slug        string         `gorm:"uniqueIndex;not null" json:"slug"`
	Description string         `gorm:"type:text" json:"description"`
	Level       int            `gorm:"default:0" json:"level"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Parent   *Category  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Products []Product  `gorm:"many2many:product_categories;" json:"products,omitempty"`
}

func (Category) TableName() string { return "categories" }

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type ProductCategory struct {
	ProductID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"product_id"`
	CategoryID uuid.UUID `gorm:"type:uuid;primaryKey" json:"category_id"`
}

func (ProductCategory) TableName() string { return "product_categories" }

type VehicleCompatibility struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	ProductID  uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	Make       string    `gorm:"not null;index" json:"make"`
	Model      string    `gorm:"not null;index" json:"model"`
	YearStart  int       `gorm:"column:year_start" json:"year_start"`
	YearEnd    int       `gorm:"column:year_end" json:"year_end"`
	Engine     string    `gorm:"" json:"engine"`
	Trim       string    `gorm:"" json:"trim"`
	Notes      string    `gorm:"type:text" json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (VehicleCompatibility) TableName() string { return "vehicle_compatibilities" }

func (v *VehicleCompatibility) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

type Tag struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Name      string         `gorm:"not null" json:"name"`
	Slug      string         `gorm:"uniqueIndex;not null" json:"slug"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Products []Product `gorm:"many2many:product_tags;" json:"products,omitempty"`
}

func (Tag) TableName() string { return "tags" }

func (t *Tag) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

type ProductTag struct {
	ProductID uuid.UUID `gorm:"type:uuid;primaryKey" json:"product_id"`
	TagID     uuid.UUID `gorm:"type:uuid;primaryKey" json:"tag_id"`
}

func (ProductTag) TableName() string { return "product_tags" }

type ProductImage struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	ProductID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"product_id"`
	URL          string         `gorm:"not null" json:"url"`
	AltText      string         `gorm:"column:alt_text" json:"alt_text"`
	DisplayOrder int            `gorm:"column:display_order;default:0" json:"display_order"`
	IsPrimary    bool           `gorm:"column:is_primary;default:false" json:"is_primary"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ProductImage) TableName() string { return "product_images" }

func (p *ProductImage) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type Specification struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	ProductID uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	SpecName  string    `gorm:"column:spec_name;not null" json:"spec_name"`
	SpecValue string    `gorm:"column:spec_value;not null" json:"spec_value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Specification) TableName() string { return "specifications" }

func (s *Specification) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
