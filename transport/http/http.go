package http

import (
	"fmt"
	"net"
	"net/http"
	"oil/config"
	"oil/docs"
	"oil/infras/postgres"
	"oil/shared/constant"
	"oil/shared/logger"
	httpMiddleware "oil/transport/http/middleware"
	"oil/transport/http/response"
	"oil/transport/http/router"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
	"github.com/rs/zerolog/log"
)

type ServerState int

const (
	ServerStateReady ServerState = iota + 1
	ServerStateInGracePeriod
	ServerStateInCleanupPeriod
)

const (
	RouteHealthCheck = "/health"
	RouteSwaggerDocs = "/swagger/*"
)

type HTTP struct {
	Config        *config.Config
	Router        router.Router
	State         ServerState
	fiber         *fiber.App
	DB            *postgres.Connection
	appMiddleware httpMiddleware.AppMiddleware
}

func New(cfg *config.Config, r router.Router, db *postgres.Connection, appMiddleware httpMiddleware.AppMiddleware) *HTTP {
	return &HTTP{
		Config:        cfg,
		Router:        r,
		DB:            db,
		appMiddleware: appMiddleware,
	}
}

func (h *HTTP) Serve() {
	h.setup()

	log.Info().Str("port", h.Config.Server.Port).Msg("Starting up HTTP server.")

	if err := h.fiber.Listen(net.JoinHostPort("0.0.0.0", h.Config.Server.Port)); err != nil {
		log.Fatal().Err(err).Msg("Failed to start HTTP server")
	}
}

func (h *HTTP) Adaptor() http.HandlerFunc {
	h.setup()

	return adaptor.FiberApp(h.fiber)
}

func (h *HTTP) setup() {
	h.setupFiber()
	h.setupMiddlewares()
	h.setupRoutes()
	h.setupSwaggerDocs()
	h.setupGracefulShutdown()
	h.State = ServerStateReady
}

func (h *HTTP) setupFiber() {
	h.fiber = fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
}

func (h *HTTP) setupRoutes() {
	h.fiber.Get(RouteHealthCheck, h.healthCheck)

	h.Router.SetupRoutes(h.fiber)
}

func (h *HTTP) setupMiddlewares() {
	h.setupServerState()
	h.setupRecover()
	h.setupLogger()
	h.setupRateLimit()
	h.setupCORS()
	h.setupTracing()
	h.logCORSConfigInfo()
}

func (h *HTTP) setupRecover() {
	h.fiber.Use(recover.New(recover.Config{
		EnableStackTrace: h.Config.Server.Env == constant.ServerEnvDevelopment,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			log.Error().
				Interface("panic", e).
				Str("path", c.Path()).
				Str("method", c.Method()).
				Str("ip", c.IP()).
				Msg("Panic recovered")
		},
	}))
}

func (h *HTTP) setupServerState() {
	h.fiber.Use(h.serverStateMiddleware())
}

func (h *HTTP) setupLogger() {
	if h.Config.Server.Env == constant.ServerEnvDevelopment {
		h.fiber.Use(fiberLogger.New(fiberLogger.Config{
			Format:     "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${error}\n",
			TimeFormat: "15:04:05",
			TimeZone:   "Local",
		}))
	} else {
		h.fiber.Use(fiberLogger.New(fiberLogger.Config{
			Format:     "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path}\n",
			TimeFormat: "2006-01-02T15:04:05Z07:00",
			TimeZone:   "Local",
		}))
	}
}

func (h *HTTP) setupTracing() {
	h.fiber.Use(h.appMiddleware.Tracing)
}

func (h *HTTP) setupRateLimit() {
	rateLimitConfig := h.Config.App.RateLimiter

	rateLimitHandler := h.appMiddleware.RateLimit()
	if rateLimitHandler != nil {
		log.Info().
			Bool("enabled", rateLimitConfig.Enable).
			Int("max_requests", rateLimitConfig.MaxRequests).
			Int("window_seconds", rateLimitConfig.WindowSeconds).
			Str("storage", "cache-redis").
			Msg("Rate limiting enabled with Redis cache storage")
		h.fiber.Use(rateLimitHandler)
	} else {
		log.Info().Msg("Rate limiting disabled")
	}
}

func (h *HTTP) setupSwaggerDocs() {
	if h.Config.Server.Env == constant.ServerEnvDevelopment {
		docs.SwaggerInfo.Title = h.Config.App.Name
		h.fiber.Get(RouteSwaggerDocs, swagger.HandlerDefault)

		return
	}
}

func (h *HTTP) setupGracefulShutdown() {
	serverStateCh := make(chan os.Signal, 1)

	signal.Notify(serverStateCh, os.Interrupt, syscall.SIGTERM)

	go h.respondToSigterm(serverStateCh)
}

func (h *HTTP) respondToSigterm(done chan os.Signal) {
	<-done

	defer os.Exit(0)

	if h.Config.Server.Env == constant.ServerEnvDevelopment {
		log.Warn().Msg("Received SIGTERM. Shutting down now.")

		return
	}

	shutdownConfig := h.Config.Server.Shutdown

	log.Info().Msg("Received SIGTERM.")
	log.Info().Int64("seconds", shutdownConfig.GracePeriodSeconds).Msg("Entering grace period.")

	h.State = ServerStateInGracePeriod

	time.Sleep(time.Duration(shutdownConfig.GracePeriodSeconds) * time.Second)

	log.Info().Int64("seconds", shutdownConfig.CleanupPeriodSeconds).Msg("Entering cleanup period.")

	h.State = ServerStateInCleanupPeriod

	time.Sleep(time.Duration(shutdownConfig.CleanupPeriodSeconds) * time.Second)

	log.Info().Msg("Cleaning up completed. Shutting down now.")
}

func (h *HTTP) serverStateMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch h.State {
		case ServerStateReady:
			// Server is ready to serve, continue normally
			return c.Next()
		case ServerStateInGracePeriod:
			// Server is in grace period. Issue a warning message and continue
			// serving as usual.
			log.Warn().Msg("SERVER IS IN GRACE PERIOD")

			return c.Next()
		case ServerStateInCleanupPeriod:
			// Server is in cleanup period. Stop the request from actually
			// invoking any domain services and respond appropriately.
			return response.WithPreparingShutdown(c)
		default:
			return c.Next()
		}
	}
}

func (h *HTTP) setupCORS() {
	corsConfig := h.Config.App.CORS
	if corsConfig.Enable {
		h.fiber.Use(cors.New(cors.Config{
			AllowOrigins:     corsConfig.AllowedOrigins,
			AllowMethods:     corsConfig.AllowedMethods,
			AllowHeaders:     corsConfig.AllowedHeaders,
			AllowCredentials: corsConfig.AllowCredentials,
			MaxAge:           corsConfig.MaxAgeSeconds,
		}))
	}
}

func (h *HTTP) logCORSConfigInfo() {
	corsConfig := h.Config.App.CORS
	corsHeaderInfo := "CORS Header"

	if corsConfig.Enable {
		log.Info().Msg("CORS Headers and Handlers are enabled.")
		log.Info().Str(corsHeaderInfo, fmt.Sprintf("Access-Control-Allow-Credentials: %t", corsConfig.AllowCredentials)).Msg("")
		log.Info().Str(corsHeaderInfo, "Access-Control-Allow-Headers: "+corsConfig.AllowedHeaders).Msg("")
		log.Info().Str(corsHeaderInfo, "Access-Control-Allow-Methods: "+corsConfig.AllowedMethods).Msg("")
		log.Info().Str(corsHeaderInfo, "Access-Control-Allow-Origin: "+corsConfig.AllowedOrigins).Msg("")
		log.Info().Str(corsHeaderInfo, fmt.Sprintf("Access-Control-Max-Age: %d", corsConfig.MaxAgeSeconds)).Msg("")
	} else {
		log.Info().Msg("CORS Headers are disabled.")
	}
}

// HealthCheck performs a health check on the server.
// @Summary Health Check
// @Description Health Check Endpoint
// @Tags service
// @Produce json
// @Accept json
// @Success 200 {object} response.Message
// @Router /health [get]
func (h *HTTP) healthCheck(c *fiber.Ctx) error {
	if err := h.DB.Read.Ping(); err != nil {
		logger.ErrorWithStack(err)

		return response.WithUnhealthy(c)
	}

	return response.WithMessage(c, fiber.StatusOK, "ok")
}
