package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/swagger"
	"github.com/rs/zerolog/log"
	"net"
	"net/http"
	"oil/config"
	"oil/docs"
	"oil/infras/postgres"
	"oil/shared/constant"
	"oil/shared/logger"
	"oil/transport/http/response"
	"oil/transport/http/router"
	"os"
	"os/signal"
	"syscall"
	"time"
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
	Config *config.Config
	Router router.Router
	State  ServerState
	fiber  *fiber.App
	DB     *postgres.Connection
}

func New(cfg *config.Config, r router.Router, db *postgres.Connection) *HTTP {
	return &HTTP{
		Config: cfg,
		Router: r,
		DB:     db,
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
	h.setupRoutes()
	h.setupSwaggerDocs()
	h.setupGracefulShutdown()
	h.State = ServerStateReady
}

func (h *HTTP) setupRoutes() {
	h.fiber = fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	h.fiber.Get(RouteHealthCheck, h.healthCheck)

	h.Router.SetupRoutes(h.fiber)
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
