package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auto-store-api/internal/middleware"
)

func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	return middleware.GetUserID(c)
}
