package handler

import (
	"github.com/gopybara/httpbara"
	"go.uber.org/fx"
)

type fxHandler struct {
	fx.Out

	Handler *httpbara.Handler `group:"handlers"`
}

func asFxHandler(h *httpbara.Handler, err error) (fxHandler, error) {
	if err != nil {
		return fxHandler{}, err
	}
	return fxHandler{Handler: h}, nil
}
