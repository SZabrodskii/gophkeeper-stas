package main

import (
	"go.uber.org/fx"

	"github.com/SZabrodskii/gophkeeper-stas/internal/config"
	"github.com/SZabrodskii/gophkeeper-stas/internal/config/db"
	"github.com/SZabrodskii/gophkeeper-stas/internal/handler"
	"github.com/SZabrodskii/gophkeeper-stas/internal/server"
	"github.com/SZabrodskii/gophkeeper-stas/pkg/logging"
)

func main() {
	fx.New(
		config.Module,
		logging.Module,
		fx.Invoke(db.NewDB),
		fx.Provide(server.NewRouter),
		fx.Invoke(
			server.StartServer,
			handler.RegisterHealthRoutes,
			handler.RegisterSignalHandler),
	).Run()
}
