//go:build wireinject
// +build wireinject

package di

import (
	"oil/config"
	"oil/infras/jwt"
	"oil/infras/otel"
	"oil/infras/postgres"
	"oil/infras/redis"
	"oil/infras/s3"
	"oil/permissions"
	"oil/shared/cache"
	"oil/transport/http"
	"oil/transport/http/middleware"
	"oil/transport/http/router"

	roomRepository "oil/internal/domains/room/repository"
	roomService "oil/internal/domains/room/service"
	roomHandler "oil/internal/handlers/room"

	bookingRepository "oil/internal/domains/booking/repository"
	bookingService "oil/internal/domains/booking/service"
	bookingHandler "oil/internal/handlers/booking"

	"github.com/google/wire"

	authService "oil/internal/domains/auth/service"
	userRepository "oil/internal/domains/user/repository"
	authHandler "oil/internal/handlers/auth"
)

var configurations = wire.NewSet(
	config.Get,
	permissions.Get,
)

var infrastructures = wire.NewSet(
	postgres.New,
	otel.New,
	redis.New,
	s3.New,
	jwt.New,
	// kafka.New,
)

var middlewares = wire.NewSet(
	middleware.NewAppMiddleware,
	middleware.NewAuthRoleMiddleware,
)

var sharedHelpers = wire.NewSet(
	cache.NewRedisCache,
)

var roomDomain = wire.NewSet(
	roomRepository.New,
	roomService.New,
)

var bookingDomain = wire.NewSet(
	bookingRepository.New,
	bookingService.New,
)

var authDomain = wire.NewSet(
	userRepository.New,
	authService.New,
)

// No galleryDomain needed

var domains = wire.NewSet(
	authDomain,
	roomDomain,
	bookingDomain,
)

var routing = wire.NewSet(
	wire.Struct(new(router.DomainHandlers), "*"),
	authHandler.New,
	roomHandler.New,
	bookingHandler.New,
	router.New,
)

func InitializeService() *http.HTTP {
	wire.Build(
		configurations,
		infrastructures,
		middlewares,
		sharedHelpers,
		domains,
		routing,
		http.New,
	)

	return &http.HTTP{}
}
