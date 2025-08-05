package middleware

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"oil/config"
	"oil/infras/otel"
	"oil/shared/constant"
)

const (
	otelHttpScopeName       = "http"
	otelMiddlewareScopeName = "middleware"
)

type AppMiddleware interface {
	Tracing(next fiber.Handler) fiber.Handler
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

func (a *appMiddleware) Tracing(next fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, scope := a.otel.NewScope(c.Context(), otelHttpScopeName, fmt.Sprintf("%s.%s.%s", otelHttpScopeName, c.Method(), c.Path()))
		defer scope.End()

		scope.SetAttributes(map[string]any{
			"app.name":        a.config.App.Name,
			"http.path":       c.Path(),
			"http.method":     c.Method(),
			"http.user_agent": c.Get(constant.ContextKeyUserAgent),
			"http.host":       c.Hostname(),
			"http.source":     c.IP(),
		})

		if err := next(c); err != nil {
			return err
		}

		scope.SetAttributes(map[string]any{
			"http.status_code": c.Response().StatusCode(),
		})

		return nil
	}
}
