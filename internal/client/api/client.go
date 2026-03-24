package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/SZabrodskii/gophkeeper-stas/internal/config"
)

// HTTPClient is the API client for communicating with the GophKeeper server.
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// NewHTTPClient creates an HTTPClient configured with TLS settings.
func NewHTTPClient(cfg *config.ClientConfig) *HTTPClient {
	tlsCfg := &tls.Config{}
	if cfg.TLSInsecure {
		tlsCfg.InsecureSkipVerify = true
	}

	return &HTTPClient{
		baseURL: cfg.ServerAddress,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsCfg,
			},
		},
	}
}

// SetToken stores the JWT token for subsequent authenticated requests.
func (c *HTTPClient) SetToken(token string) {
	c.token = token
}

// AuthRequest is the payload for register/login endpoints.
type AuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// TokenResponse is the server response containing a JWT token.
type TokenResponse struct {
	Token string `json:"token"`
}

// CreateEntryRequest is the payload for creating or updating an entry.
type CreateEntryRequest struct {
	EntryType string           `json:"entry_type"`
	Name      string           `json:"name"`
	Metadata  *json.RawMessage `json:"metadata,omitempty"`
	Data      json.RawMessage  `json:"data"`
}

// CreateEntryResponse is returned after successfully creating an entry.
type CreateEntryResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
}

// UpdateEntryResponse is returned after successfully updating an entry.
type UpdateEntryResponse struct {
	ID        string `json:"id"`
	UpdatedAt string `json:"updated_at"`
}

// EntryListItem represents a single entry in the list response.
type EntryListItem struct {
	ID        string           `json:"id"`
	EntryType string           `json:"entry_type"`
	Name      string           `json:"name"`
	Metadata  *json.RawMessage `json:"metadata,omitempty"`
	CreatedAt string           `json:"created_at"`
	UpdatedAt string           `json:"updated_at"`
}

// EntryResponse is the full entry detail returned by get/sync endpoints.
type EntryResponse struct {
	ID        string           `json:"id"`
	EntryType string           `json:"entry_type"`
	Name      string           `json:"name"`
	Metadata  *json.RawMessage `json:"metadata,omitempty"`
	Data      json.RawMessage  `json:"data,omitempty"`
	CreatedAt string           `json:"created_at"`
	UpdatedAt string           `json:"updated_at"`
}

// SyncResponse contains entries updated since the requested timestamp.
type SyncResponse struct {
	Entries    []EntryResponse `json:"entries"`
	ServerTime string          `json:"server_time"`
}

// Register sends a registration request and returns the token response.
func (c *HTTPClient) Register(ctx context.Context, login, password string) (*TokenResponse, error) {
	var resp TokenResponse
	err := c.doRequest(ctx, http.MethodPost, "/api/v1/auth/register",
		AuthRequest{Login: login, Password: password}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Login authenticates and returns the token response.
func (c *HTTPClient) Login(ctx context.Context, login, password string) (*TokenResponse, error) {
	var resp TokenResponse
	err := c.doRequest(ctx, http.MethodPost, "/api/v1/auth/login",
		AuthRequest{Login: login, Password: password}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateEntry sends a create-entry request to the server.
func (c *HTTPClient) CreateEntry(ctx context.Context, req CreateEntryRequest) (*CreateEntryResponse, error) {
	var resp CreateEntryResponse
	if err := c.doRequest(ctx, http.MethodPost, "/api/v1/entries", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListEntries fetches the list of all entries for the authenticated user.
func (c *HTTPClient) ListEntries(ctx context.Context) ([]EntryListItem, error) {
	var resp []EntryListItem
	if err := c.doRequest(ctx, http.MethodGet, "/api/v1/entries", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetEntry fetches a single entry by ID.
func (c *HTTPClient) GetEntry(ctx context.Context, id string) (*EntryResponse, error) {
	var resp EntryResponse
	if err := c.doRequest(ctx, http.MethodGet, "/api/v1/entries/"+id, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateEntry sends an update-entry request to the server.
func (c *HTTPClient) UpdateEntry(ctx context.Context, id string, req CreateEntryRequest) (*UpdateEntryResponse, error) {
	var resp UpdateEntryResponse
	if err := c.doRequest(ctx, http.MethodPut, "/api/v1/entries/"+id, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteEntry deletes an entry by ID.
func (c *HTTPClient) DeleteEntry(ctx context.Context, id string) error {
	return c.doRequest(ctx, http.MethodDelete, "/api/v1/entries/"+id, nil, nil)
}

// Sync fetches entries updated after the given timestamp.
func (c *HTTPClient) Sync(ctx context.Context, since time.Time) (*SyncResponse, error) {
	var resp SyncResponse
	path := "/api/v1/sync?since=" + since.Format(time.RFC3339)
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

type apiError struct {
	Error string `json:"error"`
}

func (c *HTTPClient) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var ae apiError
		if json.Unmarshal(respBody, &ae) == nil && ae.Error != "" {
			if resp.StatusCode == http.StatusUnauthorized {
				return fmt.Errorf("server error (%d): %s (token may be expired, try 'gophkeeper login')", resp.StatusCode, ae.Error)
			}
			return fmt.Errorf("server error (%d): %s", resp.StatusCode, ae.Error)
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("server error (%d) (token may be expired, try 'gophkeeper login')", resp.StatusCode)
		}
		return fmt.Errorf("server error (%d)", resp.StatusCode)
	}

	if resp.StatusCode == http.StatusNoContent || result == nil {
		return nil
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	return nil
}
