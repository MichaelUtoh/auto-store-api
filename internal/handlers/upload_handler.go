package handlers

import (
	"auto-store-api/internal/utils"
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"auto-store-api/pkg/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadHandler handles file uploads to S3 (or other Storage).
type UploadHandler struct {
	store        storage.Storage
	allowedTypes map[string]bool
	maxSize      int64
}

// NewUploadHandler returns an UploadHandler. store may be nil if S3 is not configured.
func NewUploadHandler(store storage.Storage, allowedTypes []string, maxSize int64) *UploadHandler {
	allowed := make(map[string]bool)
	for _, t := range allowedTypes {
		allowed[strings.TrimSpace(strings.ToLower(t))] = true
	}
	return &UploadHandler{store: store, allowedTypes: allowed, maxSize: maxSize}
}

// UploadImagesResponse is the JSON response for successful image uploads.
type UploadImagesResponse struct {
	URLs []string `json:"urls"`
}

// UploadImages godoc
// @Summary Upload image(s) to S3 (Admin/Vendor)
// @Tags upload
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Image file(s); form key: file (multiple allowed)"
// @Success 201 {object} handlers.UploadImagesResponse
// @Failure 400,413,503 {object} utils.APIResponse
// @Router /api/v1/upload/images [post]
func (h *UploadHandler) UploadImages(c *gin.Context) {
	if h.store == nil {
		utils.JSONError(c, http.StatusServiceUnavailable, "image upload is not configured (S3_BUCKET required)")
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		utils.JSONBadRequest(c, "expected multipart form with file(s)")
		return
	}

	files := form.File["file"]
	if len(files) == 0 {
		files = form.File["files"]
	}
	if len(files) == 0 {
		utils.JSONBadRequest(c, "no file(s) provided; use form key 'file' or 'files'")
		return
	}

	var urls []string
	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	for _, fh := range files {
		ct := fh.Header.Get("Content-Type")
		if ct == "" {
			ct = "application/octet-stream"
		}
		ct = strings.TrimSpace(strings.ToLower(ct))
		if !h.allowedTypes[ct] {
			utils.JSONError(c, http.StatusBadRequest, "disallowed file type: "+ct)
			return
		}
		if h.maxSize > 0 && fh.Size > h.maxSize {
			utils.JSONError(c, http.StatusRequestEntityTooLarge, "file too large: "+fh.Filename)
			return
		}

		ext := filepath.Ext(fh.Filename)
		if ext == "" {
			ext = extensionFromContentType(ct)
		}
		key := "products/" + uuid.New().String() + ext

		file, err := fh.Open()
		if err != nil {
			utils.JSONInternal(c, "failed to read uploaded file")
			return
		}
		url, err := h.store.Upload(ctx, key, file, ct, fh.Size)
		_ = file.Close()
		if err != nil {
			utils.JSONError(c, http.StatusInternalServerError, "upload failed: "+err.Error())
			return
		}
		urls = append(urls, url)
	}

	utils.JSON(c, http.StatusCreated, UploadImagesResponse{URLs: urls})
}

func extensionFromContentType(ct string) string {
	switch {
	case strings.Contains(ct, "jpeg") || strings.Contains(ct, "jpg"):
		return ".jpg"
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "webp"):
		return ".webp"
	case strings.Contains(ct, "gif"):
		return ".gif"
	default:
		return ".bin"
	}
}
