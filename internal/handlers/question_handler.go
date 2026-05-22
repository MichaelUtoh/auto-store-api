package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/middleware"
	"auto-store-api/internal/models"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type QuestionHandler struct {
	questions *services.QuestionService
}

func NewQuestionHandler(questions *services.QuestionService) *QuestionHandler {
	return &QuestionHandler{questions: questions}
}

func (h *QuestionHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	params := services.QuestionListParams{
		Query: c.Query("q"),
		Make:  c.Query("make"),
		Model: c.Query("model"),
		Page:  page,
		Limit: limit,
	}

	if pid := c.Query("product_id"); pid != "" {
		id, err := uuid.Parse(pid)
		if err != nil {
			utils.JSONBadRequest(c, "invalid product_id")
			return
		}
		params.ProductID = &id
	}
	if cid := c.Query("category_id"); cid != "" {
		id, err := uuid.Parse(cid)
		if err != nil {
			utils.JSONBadRequest(c, "invalid category_id")
			return
		}
		params.CategoryID = &id
	}
	if y := c.Query("year"); y != "" {
		year, err := strconv.Atoi(y)
		if err != nil {
			utils.JSONBadRequest(c, "invalid year")
			return
		}
		params.Year = &year
	}
	if st := c.Query("status"); st != "" {
		status := models.QuestionStatus(st)
		if !models.IsValidQuestionStatus(status) {
			utils.JSONBadRequest(c, "invalid status")
			return
		}
		params.Status = &status
	}

	questions, total, err := h.questions.List(params)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.QuestionListItemResponse, len(questions))
	for i := range questions {
		resp[i] = dto.QuestionToListItem(&questions[i])
	}
	utils.JSONPaginated(c, resp, page, limit, total)
}

func (h *QuestionHandler) GetBySlug(c *gin.Context) {
	slug := c.Param("slug")
	q, err := h.questions.GetBySlug(slug, true)
	if err != nil {
		if errors.Is(err, services.ErrQuestionNotFound) {
			utils.JSONNotFound(c, "question not found")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.QuestionToDetailResponse(q))
}

func (h *QuestionHandler) ListByProductID(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid product id")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	questions, total, err := h.questions.List(services.QuestionListParams{
		ProductID: &productID,
		Page:      page,
		Limit:     limit,
	})
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	resp := make([]dto.QuestionListItemResponse, len(questions))
	for i := range questions {
		resp[i] = dto.QuestionToListItem(&questions[i])
	}
	utils.JSONPaginated(c, resp, page, limit, total)
}

func (h *QuestionHandler) Create(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	var req dto.CreateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}

	q, err := h.questions.Create(userID, services.CreateQuestionInput{
		Title:      req.Title,
		Body:       req.Body,
		ProductID:  req.ProductID,
		CategoryID: req.CategoryID,
		Make:       req.Make,
		Model:      req.Model,
		Year:       req.Year,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidQuestionContext),
			errors.Is(err, services.ErrProductNotFound),
			errors.Is(err, services.ErrCategoryNotFound):
			utils.JSONBadRequest(c, err.Error())
		default:
			utils.JSONInternal(c, err.Error())
		}
		return
	}
	utils.JSON(c, http.StatusCreated, dto.QuestionToDetailResponse(q))
}

func (h *QuestionHandler) PostAnswer(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	questionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid question id")
		return
	}
	var req dto.CreateAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}

	answer, err := h.questions.PostAnswer(c.Request.Context(), questionID, userID, req.Body, true)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrQuestionNotFound), errors.Is(err, services.ErrAnswerNotFound):
			utils.JSONNotFound(c, err.Error())
		case errors.Is(err, services.ErrQuestionClosed), errors.Is(err, services.ErrAlreadyAnsweredByUser):
			utils.JSONBadRequest(c, err.Error())
		default:
			utils.JSONInternal(c, err.Error())
		}
		return
	}
	utils.JSON(c, http.StatusCreated, dto.AnswerToResponse(answer))
}

func (h *QuestionHandler) AcceptAnswer(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	questionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid question id")
		return
	}
	answerID, err := uuid.Parse(c.Param("answerId"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid answer id")
		return
	}

	q, err := h.questions.AcceptAnswer(questionID, answerID, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrQuestionNotFound), errors.Is(err, services.ErrAnswerNotFound):
			utils.JSONNotFound(c, err.Error())
		case errors.Is(err, services.ErrNotQuestionAuthor), errors.Is(err, services.ErrQuestionClosed):
			utils.JSONForbidden(c, err.Error())
		default:
			utils.JSONInternal(c, err.Error())
		}
		return
	}
	utils.JSON(c, http.StatusOK, dto.QuestionToDetailResponse(q))
}

func (h *QuestionHandler) Close(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	questionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.JSONBadRequest(c, "invalid question id")
		return
	}

	isAdmin := false
	if user, ok := middleware.GetUser(c); ok && user.Role == models.RoleAdmin {
		isAdmin = true
	}

	q, err := h.questions.Close(questionID, userID, isAdmin)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrQuestionNotFound):
			utils.JSONNotFound(c, err.Error())
		case errors.Is(err, services.ErrNotQuestionAuthor):
			utils.JSONForbidden(c, err.Error())
		default:
			utils.JSONInternal(c, err.Error())
		}
		return
	}
	utils.JSON(c, http.StatusOK, dto.QuestionToDetailResponse(q))
}
