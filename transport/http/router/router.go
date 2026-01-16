package router

import (
	"oil/internal/handlers/auth"
	"oil/internal/handlers/booking"
	"oil/internal/handlers/room"

	"github.com/go-chi/chi/v5"
)

type DomainHandlers struct {
	Auth    auth.Handler
	Room    room.Handler
	Booking booking.Handler
}

type Router struct {
	DomainHandlers DomainHandlers
}

func (r *Router) SetupRoutes(router chi.Router) {
	router.Route("/v1", func(routerGroup chi.Router) {
		r.DomainHandlers.Auth.Router(routerGroup)
		r.DomainHandlers.Room.Router(routerGroup)
		r.DomainHandlers.Booking.Router(routerGroup)
	})
}

func New(domainHandlers DomainHandlers) Router {
	return Router{
		DomainHandlers: domainHandlers,
	}
}
