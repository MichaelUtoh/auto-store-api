package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/middleware"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PartIdentificationHandler struct {
	partID *services.PartIdentificationService
}

func NewPartIdentificationHandler(partID *services.PartIdentificationService) *PartIdentificationHandler {
	return &PartIdentificationHandler{partID: partID}
}

func (h *PartIdentificationHandler) Identify(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		utils.JSONBadRequest(c, "expected multipart form with image file")
		return
	}

	files := form.File["image"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	if len(files) == 0 {
		utils.JSONBadRequest(c, "image file required (form key: image or file)")
		return
	}
	fh := files[0]

	make := strings.TrimSpace(c.PostForm("make"))
	model := strings.TrimSpace(c.PostForm("model"))
	yearStr := strings.TrimSpace(c.PostForm("year"))
	if make == "" || model == "" || yearStr == "" {
		utils.JSONBadRequest(c, "make, model, and year are required")
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		utils.JSONBadRequest(c, "invalid year")
		return
	}

	var labels []string
	if raw := strings.TrimSpace(c.PostForm("labels")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &labels); err != nil {
			labels = strings.Split(raw, ",")
			for i := range labels {
				labels[i] = strings.TrimSpace(labels[i])
			}
		}
	}

	file, err := fh.Open()
	if err != nil {
		utils.JSONInternal(c, "failed to read image")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		utils.JSONInternal(c, "failed to read image")
		return
	}

	ct := fh.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}

	ctx := c.Request.Context()
	imageURL, err := h.partID.UploadImage(ctx, data, ct)
	if err != nil {
		if strings.Contains(err.Error(), "not configured") {
			utils.JSONError(c, http.StatusServiceUnavailable, err.Error())
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}

	var userID *uuid.UUID
	if uid, ok := middleware.GetUserID(c); ok {
		userID = &uid
	}

	result, err := h.partID.Identify(services.PartIdentificationInput{
		UserID:     userID,
		ImageURL:   imageURL,
		Make:       make,
		Model:      model,
		Year:       year,
		SystemHint: c.PostForm("system"),
		Labels:     labels,
	})
	if err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}

	utils.JSON(c, http.StatusOK, dto.PartIdentificationToResponse(result))
}
