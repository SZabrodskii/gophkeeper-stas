package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/SZabrodskii/gophkeeper-stas/internal/service"
)

var AuthModule = fx.Module("handler.auth",
	fx.Provide(NewAuthHandler),
	fx.Invoke(RegisterAuthRoutes),
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	token, err := h.authService.Register(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrUserExists):
			c.JSON(http.StatusConflict, gin.H{"error": "login already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, tokenResponse{Token: token})
}

func RegisterAuthRoutes(router *gin.Engine, handler *AuthHandler) {
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", handler.Register)
	}
}
