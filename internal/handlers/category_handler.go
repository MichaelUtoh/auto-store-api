package handlers

import (
	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"
	"auto-store-api/internal/validators"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CategoryHandler struct {
	category *services.CategoryService
	product  *services.ProductService
}

func NewCategoryHandler(category *services.CategoryService, product *services.ProductService) *CategoryHandler {
	return &CategoryHandler{category: category, product: product}
}

// ListCategories godoc
// @Summary List all categories (tree)
// @Tags categories
// @Produce json
// @Success 200 {array} object
// @Router /api/v1/categories [get]
func (h *CategoryHandler) List(c *gin.Context) {
	categories, err := h.category.ListTree()
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, categories)
}

// GetCategory godoc
// @Summary Get category by ID
// @Tags categories
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} object
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/categories/{id} [get]
func (h *CategoryHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid category id")
		return
	}
	cat, err := h.category.GetByID(id)
	if err != nil {
		utils.JSONNotFound(c, "category not found")
		return
	}
	utils.JSON(c, http.StatusOK, cat)
}

// GetCategoryProducts godoc
// @Summary Get products in category
// @Tags categories
// @Produce json
// @Param id path string true "Category ID"
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} utils.APIResponse
// @Router /api/v1/categories/{id}/products [get]
func (h *CategoryHandler) GetProducts(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid category id")
		return
	}
	productIDs, err := h.category.GetCategoryProductIDs(id, true)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	products, total, err := h.product.ListByCategoryIDs(productIDs, page, limit)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSONPaginated(c, products, page, limit, total)
}

// CreateCategory godoc
// @Summary Create category (Admin)
// @Tags categories
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body dto.CreateCategoryRequest true "Category data"
// @Success 201 {object} object
// @Failure 400 {object} utils.APIResponse
// @Router /api/v1/categories [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	var req dto.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if req.Slug == "" {
		req.Slug = validators.Slugify(req.Name)
	}
	cat, err := h.category.Create(req.Name, req.Slug, req.Description, req.ParentID)
	if err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusCreated, cat)
}

// UpdateCategory godoc
// @Summary Update category (Admin)
// @Tags categories
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param body body dto.UpdateCategoryRequest true "Category data"
// @Success 200 {object} object
// @Failure 400,404 {object} utils.APIResponse
// @Router /api/v1/categories/{id} [put]
func (h *CategoryHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid category id")
		return
	}
	cat, err := h.category.GetByID(id)
	if err != nil {
		utils.JSONNotFound(c, "category not found")
		return
	}
	var req dto.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if req.Name != nil {
		cat.Name = *req.Name
	}
	if req.Slug != nil {
		cat.Slug = *req.Slug
	}
	if req.Description != nil {
		cat.Description = *req.Description
	}
	if req.ParentID != nil {
		cat.ParentID = req.ParentID
	}
	if err := h.category.Update(cat); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, cat)
}

// DeleteCategory godoc
// @Summary Delete category (Admin)
// @Tags categories
// @Security BearerAuth
// @Param id path string true "Category ID"
// @Success 204
// @Failure 404 {object} utils.APIResponse
// @Router /api/v1/categories/{id} [delete]
func (h *CategoryHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid category id")
		return
	}
	if err := h.category.Delete(id); err != nil {
		utils.JSONNotFound(c, "category not found")
		return
	}
	c.Status(http.StatusNoContent)
}
