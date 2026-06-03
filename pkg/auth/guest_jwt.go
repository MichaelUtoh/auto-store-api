package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const GuestTokenType = "guest"

var ErrInvalidGuestToken = errors.New("invalid guest token")

type GuestClaims struct {
	GuestID   uuid.UUID `json:"guest_id"`
	TokenType string    `json:"token_type"`
	jwt.RegisteredClaims
}

type GuestTokenManager struct {
	Secret []byte
	Expiry time.Duration
}

func NewGuestTokenManager(secret string, expiry time.Duration) *GuestTokenManager {
	return &GuestTokenManager{
		Secret: []byte(secret),
		Expiry: expiry,
	}
}

func (m *GuestTokenManager) Generate(guestID uuid.UUID) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.Expiry)
	claims := &GuestClaims{
		GuestID:   guestID,
		TokenType: GuestTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.Secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func (m *GuestTokenManager) Validate(tokenString string) (*GuestClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &GuestClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidGuestToken
		}
		return m.Secret, nil
	})
	if err != nil {
		return nil, ErrInvalidGuestToken
	}
	claims, ok := token.Claims.(*GuestClaims)
	if !ok || !token.Valid || claims.TokenType != GuestTokenType {
		return nil, ErrInvalidGuestToken
	}
	return claims, nil
}
