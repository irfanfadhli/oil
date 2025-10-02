package router

import (
	"github.com/go-chi/chi/v5"
	"oil/internal/handlers/auth"
	"oil/internal/handlers/todo"
)

type DomainHandlers struct {
	Todo todo.Handler
	Auth auth.Handler
}

type Router struct {
	DomainHandlers DomainHandlers
}

func (r *Router) SetupRoutes(router chi.Router) {
	router.Route("/v1", func(routerGroup chi.Router) {
		r.DomainHandlers.Todo.Router(routerGroup)
		r.DomainHandlers.Auth.Router(routerGroup)
	})
}

func New(domainHandlers DomainHandlers) Router {
	return Router{
		DomainHandlers: domainHandlers,
	}
}
