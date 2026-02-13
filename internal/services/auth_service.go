package services

import (
	"context"
	"errors"
	"time"

	"auto-store-api/internal/models"
	"auto-store-api/internal/repositories"
	"auto-store-api/pkg/auth"
	"auto-store-api/pkg/cache"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailExists        = errors.New("email already registered")
	ErrAccountLocked      = errors.New("account temporarily locked due to failed login attempts")
)

const (
	lockoutKeyPrefix = "auth:lockout:"
	sessionKeyPrefix = "auth:session:"
	lockoutTTL       = 15 * time.Minute
)

type AuthService struct {
	userRepo *repositories.UserRepository
	jwt      *auth.JWTManager
	cfg      AuthConfig
	log      *zap.Logger
}

type AuthConfig struct {
	LockoutAttempts int
	LockoutDuration time.Duration
}

func NewAuthService(userRepo *repositories.UserRepository, jwt *auth.JWTManager, cfg AuthConfig, log *zap.Logger) *AuthService {
	return &AuthService{userRepo: userRepo, jwt: jwt, cfg: cfg, log: log}
}

func (s *AuthService) Register(ctx context.Context, email, password, firstName, lastName, phone string) (*models.User, error) {
	exists, err := s.userRepo.ExistsByEmail(email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	count, err := s.userRepo.Count()
	if err != nil {
		return nil, err
	}
	role := models.RoleCustomer
	if count == 0 {
		role = models.RoleAdmin
	}
	user := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		FirstName:    firstName,
		LastName:     lastName,
		Phone:        phone,
		Role:         role,
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, string, string, time.Time, error) {
	lockKey := lockoutKeyPrefix + email
	if n, _ := cache.Client.Incr(ctx, lockKey).Result(); n == 1 {
		cache.Client.Expire(ctx, lockKey, s.cfg.LockoutDuration)
	}
	n, _ := cache.Client.Get(ctx, lockKey).Int()
	if n > s.cfg.LockoutAttempts {
		return nil, "", "", time.Time{}, ErrAccountLocked
	}

	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", "", time.Time{}, ErrInvalidCredentials
		}
		return nil, "", "", time.Time{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", "", time.Time{}, ErrInvalidCredentials
	}
	cache.Client.Del(ctx, lockKey)

	accessToken, _ := s.jwt.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	refreshToken, _ := s.jwt.GenerateRefreshToken(user.ID, user.Email, string(user.Role))
	expiresAt := time.Now().Add(s.jwt.AccessExpiry)
	return user, accessToken, refreshToken, expiresAt, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*models.User, string, string, time.Time, error) {
	claims, err := s.jwt.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, "", "", time.Time{}, err
	}
	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return nil, "", "", time.Time{}, err
	}
	accessToken, _ := s.jwt.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	newRefresh, _ := s.jwt.GenerateRefreshToken(user.ID, user.Email, string(user.Role))
	expiresAt := time.Now().Add(s.jwt.AccessExpiry)
	return user, accessToken, newRefresh, expiresAt, nil
}

func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	key := sessionKeyPrefix + userID.String()
	return cache.Delete(ctx, key)
}

func (s *AuthService) ForgotPassword(ctx context.Context, email string) (token string, err error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil || user == nil {
		return "", nil
	}
	token = uuid.New().String()
	key := "password_reset:" + token
	val := user.ID.String()
	if err := cache.Set(ctx, key, val, 1*time.Hour); err != nil {
		return "", err
	}
	return token, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	key := "password_reset:" + token
	var userID string
	if err := cache.Get(ctx, key, &userID); err != nil {
		return errors.New("invalid or expired token")
	}
	uid, _ := uuid.Parse(userID)
	user, err := s.userRepo.GetByID(uid)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	if err := s.userRepo.Update(user); err != nil {
		return err
	}
	cache.Delete(ctx, key)
	return nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	key := "email_verify:" + token
	var userID string
	if err := cache.Get(ctx, key, &userID); err != nil {
		return errors.New("invalid or expired token")
	}
	uid, _ := uuid.Parse(userID)
	user, err := s.userRepo.GetByID(uid)
	if err != nil {
		return err
	}
	user.EmailVerified = true
	if err := s.userRepo.Update(user); err != nil {
		return err
	}
	cache.Delete(ctx, key)
	return nil
}
