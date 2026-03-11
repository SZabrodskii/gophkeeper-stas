package handler

import (
	"github.com/gopybara/httpbara"
	"go.uber.org/fx"
)

type FxHandler struct {
	fx.Out

	Handler *httpbara.Handler `group:"handlers"`
}

func asFxHandler(h *httpbara.Handler, err error) (FxHandler, error) {
	if err != nil {
		return FxHandler{}, err
	}
	return FxHandler{Handler: h}, nil
}
