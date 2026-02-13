package middleware

import (
	"auto-store-api/internal/models"
	"auto-store-api/pkg/auth"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	UserIDKey   = "user_id"
	UserKey     = "user"
	ClaimsKey   = "claims"
)

// AuthRequired validates JWT and sets user context
func AuthRequired(jwt *auth.JWTManager, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}
		claims, err := jwt.ValidateAccessToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		var user models.User
		if err := db.First(&user, "id = ?", claims.UserID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		c.Set(ClaimsKey, claims)
		c.Set(UserIDKey, user.ID)
		c.Set(UserKey, &user)
		c.Next()
	}
}

// RequireRole checks that the authenticated user has one of the allowed roles
func RequireRole(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userVal, exists := c.Get(UserKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		user := userVal.(*models.User)
		for _, r := range roles {
			if user.Role == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}

// OptionalAuth parses JWT if present but does not require it
func OptionalAuth(jwt *auth.JWTManager, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			c.Next()
			return
		}
		claims, err := jwt.ValidateAccessToken(token)
		if err != nil {
			c.Next()
			return
		}
		var user models.User
		if err := db.First(&user, "id = ?", claims.UserID).Error; err == nil {
			c.Set(ClaimsKey, claims)
			c.Set(UserIDKey, user.ID)
			c.Set(UserKey, &user)
		}
		c.Next()
	}
}

func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	id, exists := c.Get(UserIDKey)
	if !exists {
		return uuid.Nil, false
	}
	return id.(uuid.UUID), true
}

func GetUser(c *gin.Context) (*models.User, bool) {
	u, exists := c.Get(UserKey)
	if !exists {
		return nil, false
	}
	return u.(*models.User), true
}

func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
