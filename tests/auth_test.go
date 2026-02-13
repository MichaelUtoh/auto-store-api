package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"auto-store-api/pkg/auth"
)

func TestJWTGenerateAndParse(t *testing.T) {
	jwt := auth.NewJWTManager("test-secret-key", 15*time.Minute, 24*time.Hour)
	userID := uuid.New()
	email := "test@example.com"
	role := "CUSTOMER"

	access, err := jwt.GenerateAccessToken(userID, email, role)
	assert.NoError(t, err)
	assert.NotEmpty(t, access)

	refresh, err := jwt.GenerateRefreshToken(userID, email, role)
	assert.NoError(t, err)
	assert.NotEmpty(t, refresh)

	claims, err := jwt.ValidateAccessToken(access)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)

	refreshClaims, err := jwt.ValidateRefreshToken(refresh)
	assert.NoError(t, err)
	assert.Equal(t, userID, refreshClaims.UserID)
}

func TestJWTInvalidToken(t *testing.T) {
	jwt := auth.NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)
	_, err := jwt.ValidateAccessToken("invalid-token")
	assert.Error(t, err)
}
