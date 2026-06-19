package handlers

import (
	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/middleware"
	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductHandler struct {
	product   *services.ProductService
	inventory *services.InventoryService
}

func NewProductHandler(product *services.ProductService, inventory *services.InventoryService) *ProductHandler {
	return &ProductHandler{product: product, inventory: inventory}
}

// ListProducts godoc
// @Summary List products
// @Tags products
// @Produce json
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Param category query string false "Category slug (filter to products in that category)"
// @Param search query string false "Search text (matches name, description, sku, part number)"
// @Param min query number false "Minimum price (inclusive)"
// @Param max query number false "Maximum price (inclusive); if min and max are both set, max must be greater than min"
// @Param sort query string false "price_asc|price_desc|newest"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Router /api/v1/products [get]
func (h *ProductHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	categorySlug := strings.TrimSpace(c.Query("category"))
	search := strings.TrimSpace(c.Query("search"))
	var minPrice, maxPrice *float64
	if s := strings.TrimSpace(c.Query("min")); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			utils.JSONBadRequest(c, "invalid min: must be a number")
			return
		}
		minPrice = &v
	}
	if s := strings.TrimSpace(c.Query("max")); s != "" {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			utils.JSONBadRequest(c, "invalid max: must be a number")
			return
		}
		maxPrice = &v
	}
	if minPrice != nil && maxPrice != nil && *maxPrice <= *minPrice {
		utils.JSONBadRequest(c, "max must be greater than min when both are provided")
		return
	}
	sort := strings.TrimSpace(c.Query("sort"))
	products, total, err := h.product.List(page, limit, categorySlug, search, sort, minPrice, maxPrice)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSONPaginated(c, dto.ProductsToResponse(products), page, limit, total)
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
	utils.JSON(c, http.StatusOK, dto.ProductToResponse(product))
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
	user, ok := middleware.GetUser(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	cond := models.ConditionNew
	if req.Condition != "" {
		cond = models.ProductCondition(req.Condition)
	}
	initialStock := req.StockQuantity
	threshold := req.LowStockThreshold
	if threshold <= 0 {
		threshold = models.DefaultLowStockThreshold
	}
	p := &models.Product{
		SKU:                req.SKU,
		Name:               req.Name,
		Description:        req.Description,
		Brand:              req.Brand,
		ManufacturerPartNo: req.ManufacturerPartNo,
		Price:              req.Price,
		CostPrice:          req.CostPrice,
		StockQuantity:      0,
		LowStockThreshold:  threshold,
		Weight:             req.Weight,
		Dimensions:         req.Dimensions,
		Condition:          cond,
		WarrantyMonths:     req.WarrantyMonths,
	}
	if user.Role == models.RoleVendor {
		p.VendorID = &user.ID
	} else if req.VendorID != nil {
		p.VendorID = req.VendorID
	}
	if err := h.product.Create(p, req.CategoryIDs, req.TagIDs); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if initialStock > 0 {
		performedBy := user.ID
		if _, err := h.inventory.AdjustStock(c.Request.Context(), nil, services.AdjustStockInput{
			ProductID:   p.ID,
			Delta:       initialStock,
			Reason:      models.StockMovementRestock,
			PerformedBy: &performedBy,
			Notes:       "initial stock",
		}); err != nil {
			utils.JSONBadRequest(c, err.Error())
			return
		}
	}
	product, _ := h.product.GetByID(p.ID)
	utils.JSON(c, http.StatusCreated, dto.ProductToResponse(product))
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
	user, ok := middleware.GetUser(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
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
		threshold := r.LowStockThreshold
		if threshold <= 0 {
			threshold = models.DefaultLowStockThreshold
		}
		in := services.CreateProductInput{
			SKU:                r.SKU,
			Name:               r.Name,
			Description:        r.Description,
			Brand:              r.Brand,
			ManufacturerPartNo: r.ManufacturerPartNo,
			Price:              r.Price,
			CostPrice:          r.CostPrice,
			StockQuantity:      0,
			LowStockThreshold:  threshold,
			Weight:             r.Weight,
			Dimensions:         r.Dimensions,
			Condition:          cond,
			WarrantyMonths:     r.WarrantyMonths,
			CategoryIDs:        r.CategoryIDs,
			TagIDs:             r.TagIDs,
		}
		if user.Role == models.RoleVendor {
			in.VendorID = &user.ID
		} else if r.VendorID != nil {
			in.VendorID = r.VendorID
		}
		inputs[i] = in
	}
	created, failed := h.product.CreateBatch(inputs)
	performedBy := user.ID
	for i := range created {
		orig := req.Products[created[i].Index]
		if orig.StockQuantity > 0 {
			if _, err := h.inventory.AdjustStock(c.Request.Context(), nil, services.AdjustStockInput{
				ProductID:   created[i].Product.ID,
				Delta:       orig.StockQuantity,
				Reason:      models.StockMovementRestock,
				PerformedBy: &performedBy,
				Notes:       "initial stock",
			}); err != nil {
				failed = append(failed, services.BatchProductFailure{
					Index:   created[i].Index,
					SKU:     orig.SKU,
					Message: err.Error(),
				})
				continue
			}
			if full, err := h.product.GetByID(created[i].Product.ID); err == nil {
				created[i].Product = full
			}
		}
	}
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
// @Param body body dto.UpdateProductRequest true "Product data (optional images replaces all when key present)"
// @Success 200 {object} object
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/products/{id} [put]
func (h *ProductHandler) Update(c *gin.Context) {
	user, ok := middleware.GetUser(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
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
	if !h.inventory.CanAccessProduct(user, product) {
		utils.JSONForbidden(c, "access denied")
		return
	}
	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	oldStock := product.StockQuantity
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
	if req.LowStockThreshold != nil {
		product.LowStockThreshold = *req.LowStockThreshold
	}
	if user.Role == models.RoleAdmin && req.VendorID != nil {
		product.VendorID = req.VendorID
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
	if req.StockQuantity != nil && *req.StockQuantity != oldStock {
		performedBy := user.ID
		if _, err := h.inventory.AdjustStock(c.Request.Context(), nil, services.AdjustStockInput{
			ProductID:   id,
			Delta:       *req.StockQuantity - oldStock,
			Reason:      models.StockMovementAdjustment,
			PerformedBy: &performedBy,
			Notes:       "product update",
		}); err != nil {
			utils.JSONBadRequest(c, err.Error())
			return
		}
	}
	if req.Images != nil {
		inputs := make([]services.AddImagesInput, len(*req.Images))
		for i := range *req.Images {
			item := (*req.Images)[i]
			if item.URL == "" {
				utils.JSONBadRequest(c, "images["+strconv.Itoa(i)+"].url is required")
				return
			}
			inputs[i] = services.AddImagesInput{
				URL:          item.URL,
				AltText:      item.AltText,
				DisplayOrder: item.DisplayOrder,
				IsPrimary:    item.IsPrimary,
			}
		}
		if err := h.product.ReplaceProductImages(id, inputs); err != nil {
			utils.JSONNotFound(c, "product not found")
			return
		}
	}
	updated, _ := h.product.GetByID(id)
	utils.JSON(c, http.StatusOK, dto.ProductToResponse(updated))
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
	utils.JSONPaginated(c, dto.ProductsToResponse(result.Products), q.Page, q.Limit, result.Total)
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
	compat, err := h.product.ListLinkedCompatibilities(id)
	if err != nil {
		utils.JSONNotFound(c, "product not found")
		return
	}
	utils.JSON(c, http.StatusOK, compat)
}

// AddCompatibilities godoc
// @Summary Add vehicle compatibilities to a product (Admin/Vendor)
// @Tags products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param body body dto.AddVehicleCompatibilitiesRequest true "compatibility_ids and/or compatibilities to create"
// @Success 201 {array} repositories.LinkedCompatibility
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
	if len(req.CompatibilityIDs) == 0 && len(req.Compatibilities) == 0 {
		utils.JSONBadRequest(c, "provide compatibility_ids and/or compatibilities")
		return
	}

	if len(req.CompatibilityIDs) > 0 {
		links := make([]services.LinkCompatibilitiesInput, len(req.CompatibilityIDs))
		for i, cid := range req.CompatibilityIDs {
			links[i] = services.LinkCompatibilitiesInput{CompatibilityID: cid}
		}
		if _, err := h.product.LinkCompatibilities(id, links); err != nil {
			utils.JSONBadRequest(c, err.Error())
			return
		}
	}

	if len(req.Compatibilities) > 0 {
		inputs := make([]services.AddCompatibilitiesInput, len(req.Compatibilities))
		for i, item := range req.Compatibilities {
			inputs[i] = services.AddCompatibilitiesInput{
				Make:          item.Make,
				Model:         item.Model,
				Generation:    item.Generation,
				YearStart:     item.YearStart,
				YearEnd:       item.YearEnd,
				Engine:        item.Engine,
				Trim:          item.Trim,
				MarketVariant: item.MarketVariant,
				Notes:         item.Notes,
				LinkNotes:     item.LinkNotes,
			}
		}
		if _, err := h.product.AddCompatibilities(id, inputs); err != nil {
			utils.JSONBadRequest(c, err.Error())
			return
		}
	}

	linked, err := h.product.ListLinkedCompatibilities(id)
	if err != nil {
		utils.JSONNotFound(c, "product not found")
		return
	}
	utils.JSON(c, http.StatusCreated, linked)
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

// DeleteProductImage godoc
// @Summary Delete one product image (Admin/Vendor)
// @Tags products
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Param imageId path string true "Product image ID"
// @Success 204
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/products/{id}/images/{imageId} [delete]
func (h *ProductHandler) DeleteProductImage(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	imageID, err := uuid.Parse(c.Param("imageId"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid image id")
		return
	}
	if err := h.product.DeleteProductImage(productID, imageID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.JSONNotFound(c, "product image not found")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
