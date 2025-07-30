package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/rs/zerolog/log"
	"net"
	"net/http"
	"oil/config"
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

type HTTP struct {
	Config *config.Config
	Router router.Router
	State  ServerState
	fiber  *fiber.App
}

func New(cfg *config.Config, r router.Router) *HTTP {
	return &HTTP{
		Config: cfg,
		Router: r,
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
	h.setupGracefulShutdown()
	h.State = ServerStateReady
}

func (h *HTTP) setupRoutes() {
	h.fiber = fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	h.Router.SetupRoutes(h.fiber)
}

func (h *HTTP) setupGracefulShutdown() {
	serverStateCh := make(chan os.Signal, 1)

	signal.Notify(serverStateCh, os.Interrupt, syscall.SIGTERM)

	go h.respondToSigterm(serverStateCh)
}

func (h *HTTP) respondToSigterm(done chan os.Signal) {
	<-done

	defer os.Exit(0)

	if h.Config.Server.Env == "development" {
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
