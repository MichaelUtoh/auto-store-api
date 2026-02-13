package handlers

import (
	"auto-store-api/internal/handlers/dto"
	"auto-store-api/internal/services"
	"auto-store-api/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register godoc
// @Summary Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.RegisterRequest true "Registration data"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} utils.APIResponse
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if verr, ok := err.(validator.ValidationErrors); ok {
			utils.JSONValidationErrors(c, verr)
			return
		}
		utils.JSONBadRequest(c, err.Error())
		return
	}
	user, err := h.auth.Register(c.Request.Context(), req.Email, req.Password, req.FirstName, req.LastName, req.Phone)
	if err != nil {
		if err == services.ErrEmailExists {
			utils.JSONError(c, http.StatusConflict, "email already registered")
			return
		}
		utils.JSONError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.JSON(c, http.StatusCreated, dto.UserToResponse(user))
}

// Login godoc
// @Summary User login
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.LoginRequest true "Login credentials"
// @Success 200 {object} dto.AuthResponse
// @Failure 401 {object} utils.APIResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	user, access, refresh, expiresAt, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if err == services.ErrInvalidCredentials {
			utils.JSONUnauthorized(c, "invalid email or password")
			return
		}
		if err == services.ErrAccountLocked {
			utils.JSONError(c, http.StatusTooManyRequests, "account temporarily locked")
			return
		}
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, dto.AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    expiresAt,
		User:         dto.UserToResponse(user),
	})
}

// Logout godoc
// @Summary User logout
// @Tags auth
// @Security BearerAuth
// @Success 204
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.JSONUnauthorized(c, "unauthorized")
		return
	}
	_ = h.auth.Logout(c.Request.Context(), userID)
	c.Status(http.StatusNoContent)
}

// Refresh godoc
// @Summary Refresh access token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.RefreshRequest true "Refresh token"
// @Success 200 {object} dto.AuthResponse
// @Failure 401 {object} utils.APIResponse
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	user, access, refresh, expiresAt, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		utils.JSONUnauthorized(c, "invalid or expired refresh token")
		return
	}
	utils.JSON(c, http.StatusOK, dto.AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    expiresAt,
		User:         dto.UserToResponse(user),
	})
}

// ForgotPassword godoc
// @Summary Request password reset
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.ForgotPasswordRequest true "Email"
// @Success 200 {object} utils.APIResponse
// @Router /api/v1/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	token, err := h.auth.ForgotPassword(c.Request.Context(), req.Email)
	if err != nil {
		utils.JSONInternal(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, gin.H{"message": "if the email exists, a reset link has been sent", "token": token})
}

// ResetPassword godoc
// @Summary Reset password with token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.ResetPasswordRequest true "Token and new password"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Router /api/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if err := h.auth.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, gin.H{"message": "password reset successful"})
}

// VerifyEmail godoc
// @Summary Verify email address
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.VerifyEmailRequest true "Verification token"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Router /api/v1/auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	if err := h.auth.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		utils.JSONBadRequest(c, err.Error())
		return
	}
	utils.JSON(c, http.StatusOK, gin.H{"message": "email verified"})
}
