package handlers

import (
	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReviewHandler struct {
	review *services.ReviewService
}

func NewReviewHandler(review *services.ReviewService) *ReviewHandler {
	return &ReviewHandler{review: review}
}

func (h *ReviewHandler) GetByProductID(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	reviews, total, err := h.review.GetByProductID(productID, page, limit)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSONPaginated(c, reviews, page, limit, total)
}

func (h *ReviewHandler) Create(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	review, err := h.review.Create(productID, userID, req.Rating, req.Title, req.Comment)
	if err != nil {
		if err == services.ErrAlreadyReviewed {
			utils.JSONBadRequest(c, "you have already reviewed this product")
			return
		}
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusCreated, review)
}
