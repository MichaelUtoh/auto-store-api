package tests

import (
	"testing"
	"time"

	"auto-store-api/pkg/auth"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGuestTokenGenerateAndValidate(t *testing.T) {
	m := auth.NewGuestTokenManager("guest-secret", time.Hour)
	guestID := uuid.New()

	token, expiresAt, err := m.Generate(guestID)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))

	claims, err := m.Validate(token)
	assert.NoError(t, err)
	assert.Equal(t, guestID, claims.GuestID)
	assert.Equal(t, auth.GuestTokenType, claims.TokenType)
}

func TestGuestTokenRejectsUserAccessToken(t *testing.T) {
	userJWT := auth.NewJWTManager("user-secret", time.Hour, time.Hour)
	guestJWT := auth.NewGuestTokenManager("guest-secret", time.Hour)

	userID := uuid.New()
	access, err := userJWT.GenerateAccessToken(userID, "a@b.com", "CUSTOMER")
	assert.NoError(t, err)

	_, err = guestJWT.Validate(access)
	assert.Error(t, err)
}
