package middleware

import (
	"fmt"
	"oil/config"
	"oil/infras/otel"
	"oil/shared/constant"

	"github.com/gofiber/fiber/v2"
)

const (
	otelHTTPScopeName = "http"
)

type AppMiddleware interface {
	Tracing(c *fiber.Ctx) error
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
