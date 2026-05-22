package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"auto-store-api/pkg/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PartIdentificationInput struct {
	UserID     *uuid.UUID
	ImageURL   string
	Make       string
	Model      string
	Year       int
	SystemHint string
	Labels     []string
}

type PartIdentificationCandidate struct {
	PartName   string           `json:"part_name"`
	Confidence float64          `json:"confidence"`
	HotspotID  *uuid.UUID       `json:"hotspot_id,omitempty"`
	DiagramID  *uuid.UUID       `json:"diagram_id,omitempty"`
	ProductIDs []uuid.UUID      `json:"product_ids"`
	Products   []models.Product `json:"-"`
}

type PartIdentificationResult struct {
	ID          uuid.UUID                     `json:"id"`
	Candidates  []PartIdentificationCandidate `json:"candidates"`
	DiagramID   *uuid.UUID                    `json:"diagram_id,omitempty"`
	ImageURL    string                        `json:"image_url"`
}

type PartIdentificationService struct {
	diagramRepo *repositories.DiagramRepository
	diagramSvc  *DiagramService
	store       storage.Storage
	db          *gorm.DB
}

func NewPartIdentificationService(
	diagramRepo *repositories.DiagramRepository,
	diagramSvc *DiagramService,
	store storage.Storage,
	db *gorm.DB,
) *PartIdentificationService {
	return &PartIdentificationService{
		diagramRepo: diagramRepo,
		diagramSvc:  diagramSvc,
		store:       store,
		db:          db,
	}
}

func (s *PartIdentificationService) UploadImage(ctx context.Context, data []byte, contentType string) (string, error) {
	if s.store == nil {
		return "", errors.New("image upload is not configured")
	}
	key := "part-identification/" + uuid.New().String() + extensionForContentType(contentType)
	return s.store.Upload(ctx, key, bytes.NewReader(data), contentType, int64(len(data)))
}

func extensionForContentType(ct string) string {
	ct = strings.ToLower(ct)
	switch {
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		return ".jpg"
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "webp"):
		return ".webp"
	default:
		return ".bin"
	}
}

func (s *PartIdentificationService) Identify(input PartIdentificationInput) (*PartIdentificationResult, error) {
	vehicleMake := strings.TrimSpace(input.Make)
	vehicleModel := strings.TrimSpace(input.Model)
	if vehicleMake == "" || vehicleModel == "" || input.Year < 1900 {
		return nil, errors.New("make, model, and valid year are required")
	}

	systemCode := strings.ToLower(strings.TrimSpace(input.SystemHint))
	filter := repositories.DiagramListFilter{
		Make:          vehicleMake,
		Model:         vehicleModel,
		Year:          &input.Year,
		PublishedOnly: true,
	}
	if systemCode != "" {
		filter.VehicleSystemCode = systemCode
	}

	diagrams, _, err := s.diagramRepo.ListDiagrams(filter, 0, 5)
	if err != nil {
		return nil, err
	}
	if len(diagrams) == 0 {
		return s.saveResult(input, nil, nil)
	}

	diagram := diagrams[0]
	var matched []models.DiagramHotspot
	confidenceBase := 0.45

	if len(input.Labels) > 0 {
		labelHits, err := s.diagramRepo.MatchHotspotsByLabels(diagram.ID, input.Labels)
		if err != nil {
			return nil, err
		}
		matched = append(matched, labelHits...)

		tax, err := s.diagramRepo.FindTaxonomiesByLabels(input.Labels)
		if err != nil {
			return nil, err
		}
		var patterns []string
		for _, t := range tax {
			if t.HotspotLabelPattern != "" {
				patterns = append(patterns, t.HotspotLabelPattern)
			}
		}
		if len(patterns) > 0 {
			patternHits, err := s.diagramRepo.MatchHotspotsByPatterns(diagram.ID, patterns)
			if err != nil {
				return nil, err
			}
			matched = append(matched, patternHits...)
		}
		confidenceBase = 0.75
	} else if systemCode != "" {
		all, err := s.diagramRepo.ListHotspotsByDiagramID(diagram.ID)
		if err != nil {
			return nil, err
		}
		matched = all
		confidenceBase = 0.5
	}

	matched = dedupeHotspots(matched)
	candidates := make([]PartIdentificationCandidate, 0, len(matched))
	year := input.Year

	for i, h := range matched {
		conf := confidenceBase
		if len(input.Labels) > 0 && labelMatchesHotspot(input.Labels, h.Label) {
			conf = 0.9
		} else if i > 0 {
			conf = confidenceBase - float64(i)*0.05
			if conf < 0.3 {
				conf = 0.3
			}
		}

		products, err := s.diagramSvc.GetHotspotProducts(diagram.ID, h.ID, &year)
		if err != nil {
			return nil, err
		}
		ids := make([]uuid.UUID, len(products))
		for j, p := range products {
			ids[j] = p.ID
		}

		hid := h.ID
		did := diagram.ID
		candidates = append(candidates, PartIdentificationCandidate{
			PartName:   h.Label,
			Confidence: conf,
			HotspotID:  &hid,
			DiagramID:  &did,
			ProductIDs: ids,
			Products:   products,
		})
	}

	did := diagram.ID
	return s.saveResult(input, &did, candidates)
}

func (s *PartIdentificationService) saveResult(
	input PartIdentificationInput,
	diagramID *uuid.UUID,
	candidates []PartIdentificationCandidate,
) (*PartIdentificationResult, error) {
	labelsJSON, _ := json.Marshal(input.Labels)
	record := &models.PartIdentification{
		UserID:         input.UserID,
		ImageURL:       input.ImageURL,
		Make:           strings.TrimSpace(input.Make),
		Model:          strings.TrimSpace(input.Model),
		Year:           input.Year,
		SystemHint:     strings.TrimSpace(input.SystemHint),
		LabelsJSON:     string(labelsJSON),
		CandidateCount: len(candidates),
	}
	if err := s.diagramRepo.CreatePartIdentification(record); err != nil {
		return nil, err
	}

	respCandidates := make([]PartIdentificationCandidate, len(candidates))
	for i, c := range candidates {
		respCandidates[i] = PartIdentificationCandidate{
			PartName:   c.PartName,
			Confidence: c.Confidence,
			HotspotID:  c.HotspotID,
			DiagramID:  c.DiagramID,
			ProductIDs: c.ProductIDs,
		}
	}

	return &PartIdentificationResult{
		ID:         record.ID,
		Candidates: respCandidates,
		DiagramID:  diagramID,
		ImageURL:   input.ImageURL,
	}, nil
}

func dedupeHotspots(in []models.DiagramHotspot) []models.DiagramHotspot {
	seen := make(map[uuid.UUID]bool)
	var out []models.DiagramHotspot
	for _, h := range in {
		if seen[h.ID] {
			continue
		}
		seen[h.ID] = true
		out = append(out, h)
	}
	return out
}

func labelMatchesHotspot(labels []string, hotspotLabel string) bool {
	hl := strings.ToLower(hotspotLabel)
	for _, l := range labels {
		l = strings.ToLower(strings.TrimSpace(l))
		if l == "" {
			continue
		}
		if strings.Contains(hl, l) || strings.Contains(l, hl) {
			return true
		}
	}
	return false
}
