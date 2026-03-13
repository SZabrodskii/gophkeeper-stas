package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gopybara/httpbara"
	"go.uber.org/fx"

	"github.com/SZabrodskii/gophkeeper-stas/internal/service"
)

const UserIDKey = "user_id"

var AuthModule = fx.Module("handler.auth",
	fx.Provide(NewAuthHandler),
)

type authHandlerRoutes struct {
	V1Auth        httpbara.Group      `group:"/api/v1/auth"`
	Register      httpbara.Route      `route:"POST /register" group:"v1auth"`
	Login         httpbara.Route      `route:"POST /login" group:"v1auth"`
	JWTMiddleware httpbara.Middleware `middleware:"jwt"`
}

type AuthHandler struct {
	authHandlerRoutes
	authService *service.AuthService
}

type authHandlerParams struct {
	fx.In

	AuthService *service.AuthService
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

func NewAuthHandler(params authHandlerParams) (FxHandler, error) {
	h := &AuthHandler{authService: params.AuthService}
	return asFxHandler(httpbara.AsHandler(h))
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

func (h *AuthHandler) Login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	token, err := h.authService.Login(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, tokenResponse{Token: token})
}

func (h *AuthHandler) JWTMiddleware(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
		return
	}

	userID, err := h.authService.ValidateToken(parts[1])
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	c.Set(UserIDKey, userID)
	c.Next()
}
