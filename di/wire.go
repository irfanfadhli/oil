//go:build wireinject
// +build wireinject

package di

import (
	"oil/config"
	testHandler "oil/internal/handlers/test"
	"oil/transport/http"
	"oil/transport/http/router"

	"github.com/google/wire"
)

var configurations = wire.NewSet(
	config.Get,
)

var routing = wire.NewSet(
	wire.Struct(new(router.DomainHandlers), "*"),
	testHandler.New,
	router.New,
)

func InitializeService() *http.HTTP {
	wire.Build(
		configurations,
		routing,
		http.New,
	)

	return &http.HTTP{}
}
