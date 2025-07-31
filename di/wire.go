//go:build wireinject
// +build wireinject

package di

import (
	"oil/config"
	"oil/infras/postgres"
	testHandler "oil/internal/handlers/test"
	"oil/transport/http"
	"oil/transport/http/router"

	"github.com/google/wire"
)

var configurations = wire.NewSet(
	config.Get,
)

var infrastructures = wire.NewSet(
	postgres.NewConnection,
)

var routing = wire.NewSet(
	wire.Struct(new(router.DomainHandlers), "*"),
	testHandler.New,
	router.New,
)

func InitializeService() *http.HTTP {
	wire.Build(
		configurations,
		infrastructures,
		routing,
		http.New,
	)

	return &http.HTTP{}
}
