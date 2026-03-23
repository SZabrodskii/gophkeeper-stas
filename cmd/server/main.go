// @title GophKeeper API
// @version 1.0
// @description Secure password manager API for storing credentials, text, binary data, and bank card information.
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @BasePath /api/v1
package main

import (
	"go.uber.org/fx"

	_ "github.com/SZabrodskii/gophkeeper-stas/docs"
	"github.com/SZabrodskii/gophkeeper-stas/internal/config"
	"github.com/SZabrodskii/gophkeeper-stas/internal/config/db"
	"github.com/SZabrodskii/gophkeeper-stas/internal/handler"
	"github.com/SZabrodskii/gophkeeper-stas/internal/repository"
	"github.com/SZabrodskii/gophkeeper-stas/internal/server"
	"github.com/SZabrodskii/gophkeeper-stas/internal/service"
	"github.com/SZabrodskii/gophkeeper-stas/pkg/logging"
)

func main() {
	fx.New(createApp()).Run()
}

func createApp() fx.Option {
	return fx.Options(
		config.Module,
		logging.Module,
		db.Module,
		fx.Provide(
			server.NewRouter,
			handler.NewHealthHandler,
		),
		repository.UserModule,
		repository.EntryModule,
		service.AuthModule,
		service.EntryModule,
		handler.AuthModule,
		handler.EntryModule,
		fx.Invoke(
			server.StartServer,
			handler.RegisterSignalHandler),
	)
}
