package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	Secret        []byte
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

func NewJWTManager(secret string, accessExpiry, refreshExpiry time.Duration) *JWTManager {
	return &JWTManager{
		Secret:        []byte(secret),
		AccessExpiry:  accessExpiry,
		RefreshExpiry: refreshExpiry,
	}
}

func (m *JWTManager) GenerateAccessToken(userID uuid.UUID, email, role string) (string, error) {
	return m.generateToken(userID, email, role, AccessToken, m.AccessExpiry)
}

func (m *JWTManager) GenerateRefreshToken(userID uuid.UUID, email, role string) (string, error) {
	return m.generateToken(userID, email, role, RefreshToken, m.RefreshExpiry)
}

func (m *JWTManager) generateToken(userID uuid.UUID, email, role string, tokenType TokenType, expiry time.Duration) (string, error) {
	claims := &Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.Secret)
}

func (m *JWTManager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.Secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (m *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := m.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != AccessToken {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (m *JWTManager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := m.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != RefreshToken {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
