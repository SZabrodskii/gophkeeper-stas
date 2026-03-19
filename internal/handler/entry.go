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
	ListEntries httpbara.Route `route:"GET /entries" group:"v1"`
	GetEntry    httpbara.Route `route:"GET /entries/:id" group:"v1"`
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
	case model.EntryTypeText:
		var text model.TextData
		if err := json.Unmarshal(req.Data, &text); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid text data"})
			return
		}
		entry.Text = &text
	case model.EntryTypeCard:
		var card model.CardData
		if err := json.Unmarshal(req.Data, &card); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card data"})
			return
		}
		entry.Card = &card
	case model.EntryTypeBinary:
		var binary model.BinaryData
		if err := json.Unmarshal(req.Data, &binary); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid binary data"})
			return
		}
		entry.Binary = &binary
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported entry type"})
		return
	}

	if err := h.entryService.Create(c.Request.Context(), entry); err != nil {
		if errors.Is(err, service.ErrValidation) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrPayloadTooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
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

type entryResponse struct {
	ID        uuid.UUID        `json:"id"`
	EntryType model.EntryType  `json:"entry_type"`
	Name      string           `json:"name"`
	Metadata  *json.RawMessage `json:"metadata,omitempty"`
	Data      interface{}      `json:"data,omitempty"`
	CreatedAt string           `json:"created_at"`
	UpdatedAt string           `json:"updated_at"`
}

type entryListItem struct {
	ID        uuid.UUID        `json:"id"`
	EntryType model.EntryType  `json:"entry_type"`
	Name      string           `json:"name"`
	Metadata  *json.RawMessage `json:"metadata,omitempty"`
	CreatedAt string           `json:"created_at"`
	UpdatedAt string           `json:"updated_at"`
}

func (h *EntryHandler) ListEntries(c *gin.Context) {
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

	entries, err := h.entryService.ListByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	result := make([]entryListItem, 0, len(entries))
	for _, e := range entries {
		result = append(result, entryListItem{
			ID:        e.ID,
			EntryType: e.EntryType,
			Name:      e.Name,
			Metadata:  e.Metadata,
			CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: e.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, result)
}

func (h *EntryHandler) GetEntry(c *gin.Context) {
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

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entry id"})
		return
	}

	entry, err := h.entryService.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "entry not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	resp := entryResponse{
		ID:        entry.ID,
		EntryType: entry.EntryType,
		Name:      entry.Name,
		Metadata:  entry.Metadata,
		CreatedAt: entry.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: entry.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	switch entry.EntryType {
	case model.EntryTypeCredential:
		if entry.Credential != nil {
			resp.Data = map[string]string{
				"login":    entry.Credential.Login,
				"password": entry.Credential.Password,
			}
		}
	case model.EntryTypeText:
		if entry.Text != nil {
			resp.Data = map[string]string{
				"content": entry.Text.Content,
			}
		}
	case model.EntryTypeCard:
		if entry.Card != nil {
			resp.Data = map[string]string{
				"number":      entry.Card.Number,
				"expiry":      entry.Card.Expiry,
				"holder_name": entry.Card.HolderName,
				"cvv":         entry.Card.CVV,
			}
		}
	case model.EntryTypeBinary:
		if entry.Binary != nil {
			resp.Data = map[string]string{
				"data":              entry.Binary.Data,
				"original_filename": entry.Binary.OriginalFilename,
			}
		}

	}
	c.JSON(http.StatusOK, resp)
}
