package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"oil/infras/otel"
	"oil/internal/domains/auth/model/dto"
	"oil/internal/domains/auth/service"
	"oil/shared/constant"
	"oil/shared/validator"
	"oil/transport/http/response"
)

type Handler struct {
	service service.Auth
	otel    otel.Otel
}

func New(service service.Auth, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(r fiber.Router) {
	r.Route("/auth", func(r fiber.Router) {
		r.Post("/register", handler.Register)
		r.Post("/login", handler.Login)
		r.Post("/refresh-token", handler.RefreshToken)
	})
}

// Register handles user registration
// @Summary Register a new user
// @Description Register a new user with the provided details.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Register Request"
// @Success 201 {object} response.Message "User registered successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/auth/register [post]
func (handler *Handler) Register(c *fiber.Ctx) error {
	ctx, scope := handler.otel.NewScope(c.UserContext(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".Register")
	defer scope.End()

	req := dto.RegisterRequest{}

	if err := validator.Validate(c, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		return response.WithError(c, err)
	}

	if err := handler.service.Register(ctx, req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to create todo")

		return response.WithError(c, err)
	}

	scope.AddEvent("User registered successfully")

	return response.WithMessage(c, fiber.StatusCreated, "User registered successfully")
}

// Login handles user login
// @Summary Login a user
// @Description Login a user with the provided credentials.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login Request"
// @Success 200 {object} dto.LoginResponse "User logged in successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/auth/login [post]
func (handler *Handler) Login(c *fiber.Ctx) error {
	ctx, scope := handler.otel.NewScope(c.UserContext(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".Login")
	defer scope.End()

	req := dto.LoginRequest{}

	if err := validator.Validate(c, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		return response.WithError(c, err)
	}

	res, err := handler.service.Login(ctx, req)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to login user")

		return response.WithError(c, err)
	}

	scope.AddEvent("User logged in successfully")

	return response.WithJSON(c, fiber.StatusOK, res)
}

// RefreshToken handles token refresh
// @Summary Refresh user token
// @Description Refresh user token using the provided refresh token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Refresh Token Request"
// @Success 200 {object} dto.RefreshTokenResponse "Token refreshed successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/auth/refresh-token [post]
func (handler *Handler) RefreshToken(c *fiber.Ctx) error {
	ctx, scope := handler.otel.NewScope(c.UserContext(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".RefreshToken")
	defer scope.End()

	req := dto.RefreshTokenRequest{}

	if err := validator.Validate(c, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		return response.WithError(c, err)
	}

	res, err := handler.service.RefreshToken(ctx, req)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to refresh token")

		return response.WithError(c, err)
	}

	scope.AddEvent("Token refreshed successfully")

	return response.WithJSON(c, fiber.StatusOK, res)
}
