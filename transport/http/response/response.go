package response

import (
	"github.com/gofiber/fiber/v2"
	"oil/shared/constant"
	"oil/shared/failure"
	"oil/shared/logger"
)

type Data[T any] struct {
	Data *T `json:"data,omitempty"`
}

type MetaData[T any] struct {
	Data     *T           `json:"data,omitempty"`
	MetaData *interface{} `json:"metadata,omitempty"`
}

type Error struct {
	Error *string `json:"error,omitempty"`
}

type Message struct {
	Message *string `json:"message,omitempty"`
}

// NoContent sends a response without any content
func NoContent(ctx *fiber.Ctx) error {
	return ctx.SendStatus(fiber.StatusNoContent)
}

// WithMessage sends a response with a simple text message
func WithMessage(ctx *fiber.Ctx, code int, message string) error {
	return ctx.Status(code).JSON(Message{Message: &message})
}

// WithJSON sends a response containing a JSON object
func WithJSON(ctx *fiber.Ctx, code int, jsonPayload interface{}) error {
	return ctx.Status(code).JSON(Data[any]{Data: &jsonPayload})
}

// WithMetadata sends a response containing a JSON object with metadata
func WithMetadata(ctx *fiber.Ctx, code int, jsonPayload interface{}, metadata interface{}) error {
	return ctx.Status(code).JSON(MetaData[any]{Data: &jsonPayload, MetaData: &metadata})
}

// WithError sends a response with an error message
func WithError(ctx *fiber.Ctx, err error) error {
	code := failure.GetCode(err)
	errMsg := err.Error()

	return response(ctx, code, Error{Error: &errMsg})
}

// WithPreparingShutdown sends a default response for when the server is preparing to shut down
func WithPreparingShutdown(ctx *fiber.Ctx) error {
	return WithMessage(ctx, fiber.StatusServiceUnavailable, constant.ResponseErrorPrepareShutdown)
}

// WithUnhealthy sends a default response for when the server is unhealthy
func WithUnhealthy(ctx *fiber.Ctx) error {
	return WithMessage(ctx, fiber.StatusServiceUnavailable, constant.ResponseErrorUnhealthy)
}

func response(ctx *fiber.Ctx, code int, payload interface{}) error {
	if payload == nil {
		return ctx.SendStatus(code)
	}

	ctx.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)

	err := ctx.Status(code).JSON(payload)
	if err != nil {
		logger.ErrorWithStack(err)

		return err
	}

	return nil
}
