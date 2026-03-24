package handler

import (
	"github.com/gopybara/httpbara"
	"go.uber.org/fx"
)

// FxHandler wraps an httpbara handler for fx group injection.
type FxHandler struct {
	fx.Out

	Handler *httpbara.Handler `group:"handlers"`
}

type errorResponse struct {
	Error string `json:"error" example:"error description"`
}

func asFxHandler(h *httpbara.Handler, err error) (FxHandler, error) {
	if err != nil {
		return FxHandler{}, err
	}
	return FxHandler{Handler: h}, nil
}
