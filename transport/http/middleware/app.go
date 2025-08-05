package middleware

import (
	"fmt"
	"oil/config"
	"oil/infras/otel"
	"oil/shared/constant"
	"oil/transport/http/response"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

const (
	otelHTTPScopeName = "http"
)

type AppMiddleware interface {
	Tracing(c *fiber.Ctx) error
	RateLimit() fiber.Handler
}

type appMiddleware struct {
	otel   otel.Otel
	config *config.Config
}

func NewAppMiddleware(otel otel.Otel, config *config.Config) AppMiddleware {
	return &appMiddleware{
		otel:   otel,
		config: config,
	}
}

func (a *appMiddleware) Tracing(c *fiber.Ctx) error {
	spanName := fmt.Sprintf("%s %s", c.Method(), c.Path())
	if c.Path() == "" {
		spanName = fmt.Sprintf("%s %s", c.Method(), c.Route().Path)
	}

	ctx, scope := a.otel.NewScope(c.UserContext(), otelHTTPScopeName, spanName)
	defer scope.End()

	c.SetUserContext(ctx)

	scope.SetAttributes(map[string]any{
		"app.name":        a.config.App.Name,
		"http.path":       c.Path(),
		"http.route":      c.Route().Path,
		"http.method":     c.Method(),
		"http.user_agent": c.Get(constant.ContextKeyUserAgent),
		"http.host":       c.Hostname(),
		"http.source":     c.IP(),
	})

	err := c.Next()

	scope.SetAttributes(map[string]any{
		"http.status_code": c.Response().StatusCode(),
	})

	if err != nil {
		scope.TraceError(err)
	}

	return err
}

func (a *appMiddleware) RateLimit() fiber.Handler {
	if !a.config.App.RateLimiter.Enable {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	return limiter.New(limiter.Config{
		Max:        a.config.App.RateLimiter.MaxRequests,
		Expiration: time.Duration(a.config.App.RateLimiter.WindowSeconds) * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached:           response.WithRequestLimitExceeded,
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
		Storage:                nil,
	})
}
