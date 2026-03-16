package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gopybara/httpbara"
	"go.uber.org/fx"

	"github.com/SZabrodskii/gophkeeper-stas/internal/model"
	"github.com/SZabrodskii/gophkeeper-stas/internal/service"
)

var EntryModule = fx.Module("handler.entry",
	fx.Provide(NewEntryHandler),
)

type entryHandlerRoutes struct {
	V1          httpbara.Group `group:"/api/v1" middlewares:"jwt"`
	CreateEntry httpbara.Route `route:"POST /entries" group:"v1"`
}

type EntryHandler struct {
	entryHandlerRoutes
	entryService *service.EntryService
}

type entryHandlerParams struct {
	fx.In

	EntryService *service.EntryService
}

type createEntryRequest struct {
	EntryType model.EntryType  `json:"entry_type"`
	Name      string           `json:"name"`
	Metadata  *json.RawMessage `json:"metadata,omitempty"`
	Data      json.RawMessage  `json:"data"`
}

type createEntryResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt string    `json:"created_at"`
}

func NewEntryHandler(params entryHandlerParams) (FxHandler, error) {
	h := &EntryHandler{entryService: params.EntryService}
	return asFxHandler(httpbara.AsHandler(h))
}

func (h *EntryHandler) CreateEntry(c *gin.Context) {
	userIDVal, exists := c.Get(UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req createEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	entry := &model.Entry{
		UserID:    userID,
		EntryType: req.EntryType,
		Name:      req.Name,
		Metadata:  req.Metadata,
	}

	switch req.EntryType {
	case model.EntryTypeCredential:
		var cred model.CredentialData
		if err := json.Unmarshal(req.Data, &cred); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential data"})
			return
		}
		entry.Credential = &cred
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported entry type"})
		return
	}

	if err := h.entryService.Create(c.Request.Context(), entry); err != nil {
		if errors.Is(err, service.ErrValidation) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, createEntryResponse{
		ID:        entry.ID,
		CreatedAt: entry.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}
