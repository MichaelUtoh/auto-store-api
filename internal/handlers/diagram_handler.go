package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DiagramHandler struct {
	diagrams *services.DiagramService
}

func NewDiagramHandler(diagrams *services.DiagramService) *DiagramHandler {
	return &DiagramHandler{diagrams: diagrams}
}

func (h *DiagramHandler) ListVehicleSystems(c *gin.Context) {
	systems, err := h.diagrams.ListVehicleSystems()
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.VehicleSystemResponse, len(systems))
	for i := range systems {
		resp[i] = dto.VehicleSystemToResponse(&systems[i])
	}
	utils.JSON(c, http.StatusOK, resp)
}

func (h *DiagramHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	params := services.DiagramListParams{
		Make:              c.Query("make"),
		Model:             c.Query("model"),
		VehicleSystemCode: c.Query("system"),
		Page:              page,
		Limit:             limit,
	}
	if y := c.Query("year"); y != "" {
		year, err := strconv.Atoi(y)
		if err != nil {
			utils.JSONBadRequest(c, "invalid year")
			return
		}
		params.Year = &year
	}

	diagrams, total, err := h.diagrams.List(params)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.DiagramListItemResponse, len(diagrams))
	for i := range diagrams {
		resp[i] = dto.DiagramToListItem(&diagrams[i])
	}
	utils.JSONPaginated(c, resp, page, limit, total)
}

func (h *DiagramHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid diagram id")
		return
	}
	includeHotspots := c.Query("include_hotspots") == "true"
	d, err := h.diagrams.GetByID(id, includeHotspots)
	if err != nil {
		if errors.Is(err, services.ErrDiagramNotFound) {
			utils.JSONNotFound(c, "diagram not found")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.DiagramToDetail(d))
}

func (h *DiagramHandler) ListHotspots(c *gin.Context) {
	diagramID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid diagram id")
		return
	}
	hotspots, err := h.diagrams.ListHotspots(diagramID)
	if err != nil {
		if errors.Is(err, services.ErrDiagramNotFound) {
			utils.JSONNotFound(c, "diagram not found")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.DiagramHotspotResponse, len(hotspots))
	for i := range hotspots {
		resp[i] = dto.DiagramHotspotToResponse(&hotspots[i])
	}
	utils.JSON(c, http.StatusOK, resp)
}

func (h *DiagramHandler) HotspotProducts(c *gin.Context) {
	diagramID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid diagram id")
		return
	}
	hotspotID, err := uuid.Parse(c.Param("hotspotId"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid hotspot id")
		return
	}
	var year *int
	if y := c.Query("year"); y != "" {
		v, err := strconv.Atoi(y)
		if err != nil {
			utils.JSONBadRequest(c, "invalid year")
			return
		}
		year = &v
	}
	products, err := h.diagrams.GetHotspotProducts(diagramID, hotspotID, year)
	if err != nil {
		if errors.Is(err, services.ErrDiagramNotFound) || errors.Is(err, services.ErrHotspotNotFound) {
			utils.JSONNotFound(c, err.Error())
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.ProductsToSummaryList(products))
}

func (h *DiagramHandler) Create(c *gin.Context) {
	var req dto.CreateDiagramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	d, err := h.diagrams.Create(services.CreateDiagramInput{
		VehicleSystemCode: req.VehicleSystemCode,
		Title:             req.Title,
		Make:              req.Make,
		Model:             req.Model,
		YearStart:         req.YearStart,
		YearEnd:           req.YearEnd,
		ImageURL:          req.ImageURL,
		SVGOverlayURL:     req.SVGOverlayURL,
		ImageWidth:        req.ImageWidth,
		ImageHeight:       req.ImageHeight,
		IsPublished:       req.IsPublished,
	})
	if err != nil {
		if errors.Is(err, services.ErrVehicleSystemNotFound) {
			utils.JSONBadRequest(c, err.Error())
			return
		}
		if errors.Is(err, services.ErrInvalidDiagramYears) {
			utils.JSONBadRequest(c, err.Error())
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusCreated, dto.DiagramToDetail(d))
}

func (h *DiagramHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid diagram id")
		return
	}
	var req dto.UpdateDiagramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	d, err := h.diagrams.Update(id, services.UpdateDiagramInput{
		Title:         req.Title,
		ImageURL:      req.ImageURL,
		SVGOverlayURL: req.SVGOverlayURL,
		ImageWidth:    req.ImageWidth,
		ImageHeight:   req.ImageHeight,
		IsPublished:   req.IsPublished,
		YearStart:     req.YearStart,
		YearEnd:       req.YearEnd,
	})
	if err != nil {
		if errors.Is(err, services.ErrDiagramNotFound) {
			utils.JSONNotFound(c, "diagram not found")
			return
		}
		if errors.Is(err, services.ErrInvalidDiagramYears) {
			utils.JSONBadRequest(c, err.Error())
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.DiagramToDetail(d))
}

func (h *DiagramHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid diagram id")
		return
	}
	if err := h.diagrams.Delete(id); err != nil {
		if errors.Is(err, services.ErrDiagramNotFound) {
			utils.JSONNotFound(c, "diagram not found")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *DiagramHandler) CreateHotspot(c *gin.Context) {
	diagramID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid diagram id")
		return
	}
	var req dto.CreateHotspotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	hs, err := h.diagrams.CreateHotspot(diagramID, services.CreateHotspotInput{
		Label:         req.Label,
		OEMPartNumber: req.OEMPartNumber,
		X:             req.X,
		Y:             req.Y,
		Width:         req.Width,
		Height:        req.Height,
		DisplayOrder:  req.DisplayOrder,
	})
	if err != nil {
		if errors.Is(err, services.ErrDiagramNotFound) {
			utils.JSONNotFound(c, "diagram not found")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusCreated, dto.DiagramHotspotToResponse(hs))
}

func (h *DiagramHandler) UpdateHotspot(c *gin.Context) {
	diagramID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid diagram id")
		return
	}
	hotspotID, err := uuid.Parse(c.Param("hotspotId"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid hotspot id")
		return
	}
	var req dto.UpdateHotspotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	hs, err := h.diagrams.UpdateHotspot(diagramID, hotspotID, services.UpdateHotspotInput{
		Label:         req.Label,
		OEMPartNumber: req.OEMPartNumber,
		X:             req.X,
		Y:             req.Y,
		Width:         req.Width,
		Height:        req.Height,
		DisplayOrder:  req.DisplayOrder,
	})
	if err != nil {
		if errors.Is(err, services.ErrHotspotNotFound) {
			utils.JSONNotFound(c, "hotspot not found")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.DiagramHotspotToResponse(hs))
}

func (h *DiagramHandler) DeleteHotspot(c *gin.Context) {
	diagramID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid diagram id")
		return
	}
	hotspotID, err := uuid.Parse(c.Param("hotspotId"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid hotspot id")
		return
	}
	if err := h.diagrams.DeleteHotspot(diagramID, hotspotID); err != nil {
		if errors.Is(err, services.ErrHotspotNotFound) {
			utils.JSONNotFound(c, "hotspot not found")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *DiagramHandler) LinkProduct(c *gin.Context) {
	diagramID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid diagram id")
		return
	}
	hotspotID, err := uuid.Parse(c.Param("hotspotId"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid hotspot id")
		return
	}
	var req dto.LinkHotspotProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if err := h.diagrams.LinkProduct(diagramID, hotspotID, req.ProductID, req.MatchType); err != nil {
		if errors.Is(err, services.ErrDiagramNotFound) || errors.Is(err, services.ErrHotspotNotFound) {
			utils.JSONNotFound(c, err.Error())
			return
		}
		if errors.Is(err, services.ErrProductNotFound) {
			utils.JSONNotFound(c, "product not found")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *DiagramHandler) UnlinkProduct(c *gin.Context) {
	diagramID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid diagram id")
		return
	}
	hotspotID, err := uuid.Parse(c.Param("hotspotId"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid hotspot id")
		return
	}
	productID, err := uuid.Parse(c.Param("productId"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	if err := h.diagrams.UnlinkProduct(diagramID, hotspotID, productID); err != nil {
		if errors.Is(err, services.ErrHotspotNotFound) {
			utils.JSONNotFound(c, "hotspot not found")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
