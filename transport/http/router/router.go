package router

import (
	"github.com/gofiber/fiber/v2"
	"oil/internal/handlers/test"
)

type DomainHandlers struct {
	Test test.Handler
}

type Router struct {
	DomainHandlers DomainHandlers
}

func (r *Router) SetupRoutes(router fiber.Router) {
	router.Route("/v1", func(routerGroup fiber.Router) {
		r.DomainHandlers.Test.Router(routerGroup)
	})
}

func New(domainHandlers DomainHandlers) Router {
	return Router{
		DomainHandlers: domainHandlers,
	}
}
