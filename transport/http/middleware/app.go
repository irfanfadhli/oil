package middleware

import (
	"errors"
	"fmt"
	"oil/config"
	"oil/infras/otel"
	"oil/shared"
	"oil/shared/cache"
	"oil/shared/constant"
	"oil/transport/http/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
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
	cache  cache.RedisCache
}

func NewAppMiddleware(otel otel.Otel, config *config.Config, cache cache.RedisCache) AppMiddleware {
	return &appMiddleware{
		otel:   otel,
		config: config,
		cache:  cache,
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

const (
	cacheKeyRateLimit = "limiter"
)

func (a *appMiddleware) RateLimit() fiber.Handler {
	if !a.config.App.RateLimiter.Enable {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	maxReqs := a.config.App.RateLimiter.MaxRequests
	windowSecs := a.config.App.RateLimiter.WindowSeconds

	return func(c *fiber.Ctx) error {
		cacheKey := shared.BuildCacheKey(cacheKeyRateLimit, c.IP())

		var count int
		err := a.cache.Get(c.UserContext(), cacheKey, &count)

		if err != nil {
			if errors.Is(err, cache.CacheNil) {
				count = 1
			} else {
				return c.Next()
			}
		} else {
			count++
		}

		if count > maxReqs {
			return response.WithRequestLimitExceeded(c)
		}

		err = a.cache.Save(c.UserContext(), cacheKey, count, windowSecs)
		if err != nil {
			return c.Next()
		}

		c.Set(constant.HeaderRateLimit, strconv.Itoa(maxReqs))
		c.Set(constant.HeaderRateLimitRemaining, strconv.Itoa(max(0, maxReqs-count)))
		c.Set(constant.HeaderRateLimitWindow, strconv.Itoa(windowSecs))

		return c.Next()
	}
}
