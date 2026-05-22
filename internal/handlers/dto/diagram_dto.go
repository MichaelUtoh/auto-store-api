package dto

import (
	"auto-store-api/internal/models"
	"auto-store-api/internal/services"

	"github.com/google/uuid"
)

type VehicleSystemResponse struct {
	ID           uuid.UUID `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	DisplayOrder int       `json:"display_order"`
}

func VehicleSystemToResponse(v *models.VehicleSystem) VehicleSystemResponse {
	return VehicleSystemResponse{
		ID:           v.ID,
		Code:         v.Code,
		Name:         v.Name,
		Description:  v.Description,
		DisplayOrder: v.DisplayOrder,
	}
}

type DiagramListItemResponse struct {
	ID            uuid.UUID             `json:"id"`
	Title         string                `json:"title"`
	Make          string                `json:"make"`
	Model         string                `json:"model"`
	YearStart     int                   `json:"year_start"`
	YearEnd       int                   `json:"year_end"`
	ImageURL      string                `json:"image_url"`
	VehicleSystem VehicleSystemResponse `json:"vehicle_system"`
}

type DiagramDetailResponse struct {
	ID            uuid.UUID               `json:"id"`
	Title         string                  `json:"title"`
	Make          string                  `json:"make"`
	Model         string                  `json:"model"`
	YearStart     int                     `json:"year_start"`
	YearEnd       int                     `json:"year_end"`
	ImageURL      string                  `json:"image_url"`
	SVGOverlayURL string                  `json:"svg_overlay_url,omitempty"`
	ImageWidth    int                     `json:"image_width"`
	ImageHeight   int                     `json:"image_height"`
	VehicleSystem VehicleSystemResponse   `json:"vehicle_system"`
	Hotspots      []DiagramHotspotResponse `json:"hotspots,omitempty"`
}

type DiagramHotspotResponse struct {
	ID            uuid.UUID `json:"id"`
	DiagramID     uuid.UUID `json:"diagram_id"`
	Label         string    `json:"label"`
	OEMPartNumber string    `json:"oem_part_number,omitempty"`
	X             float64   `json:"x"`
	Y             float64   `json:"y"`
	Width         float64   `json:"width"`
	Height        float64   `json:"height"`
	DisplayOrder  int       `json:"display_order"`
}

type ProductSummaryResponse struct {
	ID                   uuid.UUID `json:"id"`
	SKU                  string    `json:"sku"`
	Name                 string    `json:"name"`
	Brand                string    `json:"brand"`
	ManufacturerPartNo   string    `json:"manufacturer_part_number"`
	Price                float64   `json:"price"`
	Condition            string    `json:"condition"`
	StockQuantity        int       `json:"stock_quantity"`
	PrimaryImageURL      string    `json:"primary_image_url,omitempty"`
}

type CreateDiagramRequest struct {
	VehicleSystemCode string `json:"vehicle_system_code" binding:"required"`
	Title             string `json:"title" binding:"required,max=200"`
	Make              string `json:"make" binding:"required,max=100"`
	Model             string `json:"model" binding:"required,max=100"`
	YearStart         int    `json:"year_start" binding:"required,min=1900"`
	YearEnd           int    `json:"year_end" binding:"required,min=1900"`
	ImageURL          string `json:"image_url" binding:"required,url"`
	SVGOverlayURL     string `json:"svg_overlay_url" binding:"omitempty,url"`
	ImageWidth        int    `json:"image_width"`
	ImageHeight       int    `json:"image_height"`
	IsPublished       bool   `json:"is_published"`
}

type UpdateDiagramRequest struct {
	Title         *string `json:"title" binding:"omitempty,max=200"`
	ImageURL      *string `json:"image_url" binding:"omitempty,url"`
	SVGOverlayURL *string `json:"svg_overlay_url" binding:"omitempty,url"`
	ImageWidth    *int    `json:"image_width"`
	ImageHeight   *int    `json:"image_height"`
	IsPublished   *bool   `json:"is_published"`
	YearStart     *int    `json:"year_start" binding:"omitempty,min=1900"`
	YearEnd       *int    `json:"year_end" binding:"omitempty,min=1900"`
}

type CreateHotspotRequest struct {
	Label         string  `json:"label" binding:"required,max=120"`
	OEMPartNumber string  `json:"oem_part_number" binding:"max=100"`
	X             float64 `json:"x" binding:"required,min=0,max=100"`
	Y             float64 `json:"y" binding:"required,min=0,max=100"`
	Width         float64 `json:"width" binding:"required,min=0,max=100"`
	Height        float64 `json:"height" binding:"required,min=0,max=100"`
	DisplayOrder  int     `json:"display_order"`
}

type UpdateHotspotRequest struct {
	Label         *string  `json:"label" binding:"omitempty,max=120"`
	OEMPartNumber *string  `json:"oem_part_number" binding:"omitempty,max=100"`
	X             *float64 `json:"x" binding:"omitempty,min=0,max=100"`
	Y             *float64 `json:"y" binding:"omitempty,min=0,max=100"`
	Width         *float64 `json:"width" binding:"omitempty,min=0,max=100"`
	Height        *float64 `json:"height" binding:"omitempty,min=0,max=100"`
	DisplayOrder  *int     `json:"display_order"`
}

type LinkHotspotProductRequest struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	MatchType string    `json:"match_type" binding:"omitempty,oneof=primary alternate"`
}

type PartIdentificationCandidateResponse struct {
	PartName   string                   `json:"part_name"`
	Confidence float64                  `json:"confidence"`
	HotspotID  *uuid.UUID               `json:"hotspot_id,omitempty"`
	DiagramID  *uuid.UUID               `json:"diagram_id,omitempty"`
	ProductIDs []uuid.UUID              `json:"product_ids"`
	Products   []ProductSummaryResponse `json:"products,omitempty"`
}

type PartIdentificationResponse struct {
	ID         uuid.UUID                             `json:"id"`
	ImageURL   string                                `json:"image_url"`
	DiagramID  *uuid.UUID                            `json:"diagram_id,omitempty"`
	Candidates []PartIdentificationCandidateResponse `json:"candidates"`
}

func DiagramToListItem(d *models.Diagram) DiagramListItemResponse {
	return DiagramListItemResponse{
		ID:            d.ID,
		Title:         d.Title,
		Make:          d.Make,
		Model:         d.Model,
		YearStart:     d.YearStart,
		YearEnd:       d.YearEnd,
		ImageURL:      d.ImageURL,
		VehicleSystem: VehicleSystemToResponse(&d.VehicleSystem),
	}
}

func DiagramToDetail(d *models.Diagram) DiagramDetailResponse {
	resp := DiagramDetailResponse{
		ID:            d.ID,
		Title:         d.Title,
		Make:          d.Make,
		Model:         d.Model,
		YearStart:     d.YearStart,
		YearEnd:       d.YearEnd,
		ImageURL:      d.ImageURL,
		SVGOverlayURL: d.SVGOverlayURL,
		ImageWidth:    d.ImageWidth,
		ImageHeight:   d.ImageHeight,
		VehicleSystem: VehicleSystemToResponse(&d.VehicleSystem),
	}
	if len(d.Hotspots) > 0 {
		resp.Hotspots = make([]DiagramHotspotResponse, len(d.Hotspots))
		for i := range d.Hotspots {
			resp.Hotspots[i] = DiagramHotspotToResponse(&d.Hotspots[i])
		}
	}
	return resp
}

func DiagramHotspotToResponse(h *models.DiagramHotspot) DiagramHotspotResponse {
	return DiagramHotspotResponse{
		ID:            h.ID,
		DiagramID:     h.DiagramID,
		Label:         h.Label,
		OEMPartNumber: h.OEMPartNumber,
		X:             h.X,
		Y:             h.Y,
		Width:         h.Width,
		Height:        h.Height,
		DisplayOrder:  h.DisplayOrder,
	}
}

func ProductToSummary(p *models.Product) ProductSummaryResponse {
	resp := ProductSummaryResponse{
		ID:                 p.ID,
		SKU:                p.SKU,
		Name:               p.Name,
		Brand:              p.Brand,
		ManufacturerPartNo: p.ManufacturerPartNo,
		Price:              p.Price,
		Condition:          string(p.Condition),
		StockQuantity:      p.StockQuantity,
	}
	for _, img := range p.Images {
		if img.IsPrimary || resp.PrimaryImageURL == "" {
			resp.PrimaryImageURL = img.URL
			if img.IsPrimary {
				break
			}
		}
	}
	return resp
}

func ProductsToSummaryList(products []models.Product) []ProductSummaryResponse {
	out := make([]ProductSummaryResponse, len(products))
	for i := range products {
		out[i] = ProductToSummary(&products[i])
	}
	return out
}

func PartIdentificationToResponse(r *services.PartIdentificationResult) PartIdentificationResponse {
	candidates := make([]PartIdentificationCandidateResponse, len(r.Candidates))
	for i, c := range r.Candidates {
		candidates[i] = PartIdentificationCandidateResponse{
			PartName:   c.PartName,
			Confidence: c.Confidence,
			HotspotID:  c.HotspotID,
			DiagramID:  c.DiagramID,
			ProductIDs: c.ProductIDs,
		}
	}
	return PartIdentificationResponse{
		ID:         r.ID,
		ImageURL:   r.ImageURL,
		DiagramID:  r.DiagramID,
		Candidates: candidates,
	}
}
