package utils

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Errors  []FieldErr  `json:"errors,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// FieldErr is a single field validation error for API responses.
type FieldErr struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Meta struct {
	Page       int   `json:"page,omitempty"`
	Limit      int   `json:"limit,omitempty"`
	Total      int64 `json:"total,omitempty"`
	TotalPages int   `json:"total_pages,omitempty"`
}

func JSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, APIResponse{Success: true, Data: data})
}

func JSONPaginated(c *gin.Context, data interface{}, page, limit int, total int64) {
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func JSONError(c *gin.Context, status int, err string) {
	c.JSON(status, APIResponse{Success: false, Error: err})
}

func JSONBadRequest(c *gin.Context, err string) {
	JSONError(c, http.StatusBadRequest, err)
}

// JSONValidationErrors responds with 400 and field-level validation messages.
func JSONValidationErrors(c *gin.Context, errs validator.ValidationErrors) {
	fields := make([]FieldErr, 0, len(errs))
	for _, e := range errs {
		fields = append(fields, FieldErr{
			Field:   strings.ToLower(e.Field()),
			Message: formatValidationTag(e.Tag(), e.Param(), e.Field()),
		})
	}
	c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Error:   "validation failed",
		Errors:  fields,
	})
}

func formatValidationTag(tag, param, field string) string {
	switch tag {
	case "required":
		return field + " is required"
	case "email":
		return "invalid email format"
	case "min":
		return field + " must be at least " + param + " characters"
	case "max":
		return field + " must be at most " + param + " characters"
	case "oneof":
		return field + " must be one of: " + param
	case "phone":
		return "invalid phone number format"
	case "slug":
		return "invalid slug format (use lowercase letters, numbers, hyphens)"
	default:
		return field + " failed validation: " + tag
	}
}

func JSONUnauthorized(c *gin.Context, err string) {
	JSONError(c, http.StatusUnauthorized, err)
}

func JSONForbidden(c *gin.Context, err string) {
	JSONError(c, http.StatusForbidden, err)
}

func JSONNotFound(c *gin.Context, err string) {
	JSONError(c, http.StatusNotFound, err)
}

func JSONInternal(c *gin.Context, err string) {
	JSONError(c, http.StatusInternalServerError, err)
}
