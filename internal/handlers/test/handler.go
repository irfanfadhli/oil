package test

import "github.com/gofiber/fiber/v2"

type Handler struct{}

func New() Handler {
	return Handler{}
}

func (h *Handler) Router(r fiber.Router) {
	r.Route("/test", func(r fiber.Router) {
		r.Get("/", h.Test)
	})
}

func (h *Handler) Test(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "test",
	})
}
