package handlers

import (
	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProductHandler struct {
	product *services.ProductService
}

func NewProductHandler(product *services.ProductService) *ProductHandler {
	return &ProductHandler{product: product}
}

// ListProducts godoc
// @Summary List products
// @Tags products
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} utils.APIResponse
// @Router /api/v1/products [get]
func (h *ProductHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	products, total, err := h.product.List(page, limit, nil)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSONPaginated(c, products, page, limit, total)
}

// GetProduct godoc
// @Summary Get product by ID
// @Tags products
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} object
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/products/{id} [get]
func (h *ProductHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	product, err := h.product.GetByID(id)
	if err != nil {
		utils.JSONNotFound(c, "product not found")
		return
	}
	utils.JSON(c, http.StatusOK, product)
}

// CreateProduct godoc
// @Summary Create product (Admin/Vendor)
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.CreateProductRequest true "Product data"
// @Success 201 {object} object
// @Failure 400 {object} utils.APIResponse
// @Router /api/v1/products [post]
func (h *ProductHandler) Create(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	cond := models.ConditionNew
	if req.Condition != "" {
		cond = models.ProductCondition(req.Condition)
	}
	p := &models.Product{
		SKU:                req.SKU,
		Name:               req.Name,
		Description:        req.Description,
		Brand:              req.Brand,
		ManufacturerPartNo: req.ManufacturerPartNo,
		Price:              req.Price,
		CostPrice:          req.CostPrice,
		StockQuantity:      req.StockQuantity,
		Weight:             req.Weight,
		Dimensions:         req.Dimensions,
		Condition:          cond,
		WarrantyMonths:     req.WarrantyMonths,
	}
	if err := h.product.Create(p, req.CategoryIDs, req.TagIDs); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	product, _ := h.product.GetByID(p.ID)
	utils.JSON(c, http.StatusCreated, product)
}

const maxBatchSize = 100

// CreateProductsBatch godoc
// @Summary Create multiple products (Admin/Vendor)
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.CreateProductsBatchRequest true "Products to create"
// @Success 201 {object} dto.CreateProductsBatchResponse
// @Failure 400 {object} utils.APIResponse
// @Router /api/v1/products/batch [post]
func (h *ProductHandler) CreateBatch(c *gin.Context) {
	var req dto.CreateProductsBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if len(req.Products) == 0 {
		utils.JSONBadRequest(c, "products array cannot be empty")
		return
	}
	if len(req.Products) > maxBatchSize {
		utils.JSONBadRequest(c, "batch size exceeds maximum of 100 products")
		return
	}
	inputs := make([]services.CreateProductInput, len(req.Products))
	for i := range req.Products {
		r := &req.Products[i]
		cond := models.ConditionNew
		if r.Condition != "" {
			cond = models.ProductCondition(r.Condition)
		}
		inputs[i] = services.CreateProductInput{
			SKU:                r.SKU,
			Name:               r.Name,
			Description:        r.Description,
			Brand:              r.Brand,
			ManufacturerPartNo: r.ManufacturerPartNo,
			Price:              r.Price,
			CostPrice:          r.CostPrice,
			StockQuantity:      r.StockQuantity,
			Weight:             r.Weight,
			Dimensions:         r.Dimensions,
			Condition:          cond,
			WarrantyMonths:     r.WarrantyMonths,
			CategoryIDs:        r.CategoryIDs,
			TagIDs:             r.TagIDs,
		}
	}
	created, failed := h.product.CreateBatch(inputs)
	resp := dto.CreateProductsBatchResponse{
		Created: make([]dto.BatchProductResult, len(created)),
		Failed:  make([]dto.BatchProductError, len(failed)),
	}
	for i := range created {
		resp.Created[i] = dto.BatchProductResult{Index: created[i].Index, Product: created[i].Product}
	}
	for i := range failed {
		resp.Failed[i] = dto.BatchProductError{
			Index:   failed[i].Index,
			SKU:     failed[i].SKU,
			Message: failed[i].Message,
		}
	}
	status := http.StatusCreated
	if len(created) == 0 && len(failed) > 0 {
		status = http.StatusBadRequest
	}
	utils.JSON(c, status, resp)
}

// UpdateProduct godoc
// @Summary Update product (Admin/Vendor)
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param body body dto.UpdateProductRequest true "Product data"
// @Success 200 {object} object
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/products/{id} [put]
func (h *ProductHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	product, err := h.product.GetByID(id)
	if err != nil {
		utils.JSONNotFound(c, "product not found")
		return
	}
	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = *req.Description
	}
	if req.Brand != nil {
		product.Brand = *req.Brand
	}
	if req.ManufacturerPartNo != nil {
		product.ManufacturerPartNo = *req.ManufacturerPartNo
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.CostPrice != nil {
		product.CostPrice = *req.CostPrice
	}
	if req.StockQuantity != nil {
		product.StockQuantity = *req.StockQuantity
	}
	if req.Weight != nil {
		product.Weight = *req.Weight
	}
	if req.Dimensions != nil {
		product.Dimensions = *req.Dimensions
	}
	if req.Condition != nil {
		product.Condition = models.ProductCondition(*req.Condition)
	}
	if req.WarrantyMonths != nil {
		product.WarrantyMonths = *req.WarrantyMonths
	}
	if err := h.product.Update(product, req.CategoryIDs, req.TagIDs); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	updated, _ := h.product.GetByID(id)
	utils.JSON(c, http.StatusOK, updated)
}

// DeleteProduct godoc
// @Summary Delete product (Admin)
// @Tags products
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Success 204
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/products/{id} [delete]
func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	if err := h.product.Delete(id); err != nil {
		utils.JSONNotFound(c, "product not found")
		return
	}
	c.Status(http.StatusNoContent)
}

// SearchProducts godoc
// @Summary Advanced product search
// @Tags products
// @Produce json
// @Param q query string false "Search query"
// @Param category query string false "Category slug"
// @Param tags query string false "Tags (comma-separated)"
// @Param make query string false "Vehicle make"
// @Param model query string false "Vehicle model"
// @Param year query string false "Year range (e.g. 2015-2020)"
// @Param minPrice query number false "Min price"
// @Param maxPrice query number false "Max price"
// @Param condition query string false "new|used|refurbished"
// @Param sort query string false "price_asc|price_desc|newest"
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} utils.APIResponse
// @Router /api/v1/products/search [get]
func (h *ProductHandler) Search(c *gin.Context) {
	var q dto.SearchProductsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	yearStart, yearEnd := repositories.ParseYearRange(q.Year)
	params := repositories.SearchParams{
		Q:         q.Q,
		Category:  q.Category,
		Tags:      q.Tags,
		Make:      q.Make,
		Model:     q.Model,
		YearStart: yearStart,
		YearEnd:   yearEnd,
		MinPrice:  q.MinPrice,
		MaxPrice:  q.MaxPrice,
		Condition: q.Condition,
		Brand:     q.Brand,
		Sort:      q.Sort,
		Page:      q.Page,
		Limit:     q.Limit,
	}
	result, err := h.product.Search(params)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSONPaginated(c, result.Products, q.Page, q.Limit, result.Total)
}

// GetCompatibility godoc
// @Summary Get vehicle compatibility for product
// @Tags products
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {array} object
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/products/{id}/compatibility [get]
func (h *ProductHandler) GetCompatibility(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	product, err := h.product.GetByID(id)
	if err != nil {
		utils.JSONNotFound(c, "product not found")
		return
	}
	utils.JSON(c, http.StatusOK, product.Compatibilities)
}

// AddCompatibilities godoc
// @Summary Add vehicle compatibilities to a product (Admin/Vendor)
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param body body dto.AddVehicleCompatibilitiesRequest true "Compatibilities (make, model, year_start, year_end, engine, trim, notes)"
// @Success 201 {array} models.VehicleCompatibility
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/products/{id}/compatibility [post]
func (h *ProductHandler) AddCompatibilities(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	var req dto.AddVehicleCompatibilitiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	inputs := make([]services.AddCompatibilitiesInput, len(req.Compatibilities))
	for i := range req.Compatibilities {
		inputs[i] = services.AddCompatibilitiesInput{
			Make:      req.Compatibilities[i].Make,
			Model:     req.Compatibilities[i].Model,
			YearStart: req.Compatibilities[i].YearStart,
			YearEnd:   req.Compatibilities[i].YearEnd,
			Engine:    req.Compatibilities[i].Engine,
			Trim:      req.Compatibilities[i].Trim,
			Notes:     req.Compatibilities[i].Notes,
		}
	}
	created, err := h.product.AddCompatibilities(id, inputs)
	if err != nil {
		utils.JSONNotFound(c, "product not found")
		return
	}
	utils.JSON(c, http.StatusCreated, created)
}

// AddImages godoc
// @Summary Add images to a product (Admin/Vendor)
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param body body dto.AddProductImagesRequest true "Images (url, optional alt_text, display_order, is_primary)"
// @Success 201 {array} models.ProductImage
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/products/{id}/images [post]
func (h *ProductHandler) AddImages(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	var req dto.AddProductImagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	inputs := make([]services.AddImagesInput, len(req.Images))
	for i := range req.Images {
		inputs[i] = services.AddImagesInput{
			URL:          req.Images[i].URL,
			AltText:      req.Images[i].AltText,
			DisplayOrder: req.Images[i].DisplayOrder,
			IsPrimary:    req.Images[i].IsPrimary,
		}
	}
	created, err := h.product.AddImages(id, inputs)
	if err != nil {
		utils.JSONNotFound(c, "product not found")
		return
	}
	utils.JSON(c, http.StatusCreated, created)
}
