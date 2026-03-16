package main

import (
	"go.uber.org/fx"

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
