package middleware

import (
	"context"
	"errors"
	"oil/infras/jwt"
	"oil/infras/otel"
	"oil/shared/constant"
	"oil/shared/failure"
	"oil/transport/http/response"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// Auth defines the interface for authentication middleware
type Auth interface {
	Auth() fiber.Handler
}

// Role defines the interface for role-based access control middleware
type Role interface {
	RBAC(requiredRoles ...string) fiber.Handler
	RequireSuperAdmin() fiber.Handler
	RequireAdmin() fiber.Handler
	RequireUser() fiber.Handler
}

// AuthRole combines all middleware interfaces
type AuthRole interface {
	Auth
	Role
}

// authRoleImpl implements the AuthRole interface
type authRoleImpl struct {
	jwtService jwt.JWT
	otel       otel.Otel
}

// NewAuthRoleMiddleware creates a new middleware instance
func NewAuthRoleMiddleware(jwtService jwt.JWT, otel otel.Otel) AuthRole {
	return &authRoleImpl{
		jwtService: jwtService,
		otel:       otel,
	}
}

// Auth validates JWT tokens
// Requires valid authentication for all requests
func (m *authRoleImpl) Auth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, scope := m.otel.NewScope(c.Context(), constant.OtelHandlerScopeName, "auth.middleware")
		defer scope.End()

		scope.SetAttributes(map[string]any{
			"middleware.type": "auth",
			"http.path":       c.Path(),
			"http.method":     c.Method(),
		})

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			scope.SetAttributes(map[string]any{
				"auth.result": "failed",
				"auth.reason": "missing_header",
			})

			log.Warn().
				Str("path", c.Path()).
				Str("method", c.Method()).
				Msg("Missing authorization header")

			return response.WithError(c, failure.Unauthorized("Missing authorization header"))
		}

		tokenString, err := jwt.ExtractTokenFromHeader(authHeader)
		if err != nil {
			scope.SetAttributes(map[string]any{
				"auth.result": "failed",
				"auth.reason": "invalid_header_format",
			})
			scope.TraceIfError(err)

			log.Warn().
				Err(err).
				Str("path", c.Path()).
				Str("method", c.Method()).
				Msg("Invalid authorization header format")

			return response.WithError(c, failure.Unauthorized("Invalid authorization header format"))
		}

		claims, err := m.jwtService.ValidateToken(tokenString, jwt.AccessToken)
		if err != nil {
			scope.SetAttributes(map[string]any{
				"auth.result": "failed",
				"auth.reason": "token_validation_failed",
			})
			scope.TraceIfError(err)

			log.Warn().
				Err(err).
				Str("path", c.Path()).
				Str("method", c.Method()).
				Msg("Invalid or expired token")

			var message string
			switch {
			case errors.Is(err, jwt.ErrExpiredToken):
				message = "Token has expired"
			case errors.Is(err, jwt.ErrInvalidToken):
				message = "Invalid token"
			case errors.Is(err, jwt.ErrInvalidClaim):
				message = "Invalid token claims"
			default:
				message = "Token validation failed"
			}

			return response.WithError(c, failure.Unauthorized(message))
		}

		// Validate that required claims are not empty
		if claims.UserID == "" {
			log.Error().Msg("JWT claims: UserID is empty")
			return response.WithError(c, failure.Unauthorized("Invalid token claims"))
		}
		if claims.Email == "" {
			log.Error().Msg("JWT claims: Email is empty")
			return response.WithError(c, failure.Unauthorized("Invalid token claims"))
		}

		// Store user information in Go context
		ctx := c.UserContext()
		ctx = context.WithValue(ctx, constant.ContextKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, constant.ContextKeyUserEmail, claims.Email)
		ctx = context.WithValue(ctx, constant.ContextKeyUserRole, claims.Role)
		ctx = context.WithValue(ctx, constant.ContextKeyTokenID, claims.TokenID)
		c.SetUserContext(ctx)

		// Also store in Fiber locals for backward compatibility
		c.Locals(constant.ContextKeyUserID, claims.UserID)
		c.Locals(constant.ContextKeyUserEmail, claims.Email)
		c.Locals(constant.ContextKeyUserRole, claims.Role)
		c.Locals(constant.ContextKeyTokenID, claims.TokenID)

		scope.SetAttributes(map[string]any{
			"auth.result":   "success",
			"auth.user_id":  claims.UserID,
			"auth.email":    claims.Email,
			"auth.role":     claims.Role,
			"auth.token_id": claims.TokenID,
		})

		return c.Next()
	}
}

// RBAC checks if user has required role(s)
// Requires prior authentication via Auth middleware
func (m *authRoleImpl) RBAC(requiredRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, scope := m.otel.NewScope(c.Context(), constant.OtelHandlerScopeName, "rbac.middleware")
		defer scope.End()

		scope.SetAttributes(map[string]any{
			"middleware.type":     "rbac",
			"rbac.required_roles": requiredRoles,
			"http.path":           c.Path(),
			"http.method":         c.Method(),
		})

		userRole := c.UserContext().Value(constant.ContextKeyUserRole)
		if userRole == nil {
			// Fallback to Fiber locals for backward compatibility
			userRole = c.Locals(constant.ContextKeyUserRole)
		}

		if userRole == nil {
			scope.SetAttributes(map[string]any{
				"rbac.result": "failed",
				"rbac.reason": "no_user_role_in_context",
			})

			log.Warn().
				Str("path", c.Path()).
				Str("method", c.Method()).
				Msg("User role not found in context")

			return response.WithError(c, failure.Unauthorized("User role not found"))
		}

		role := userRole.(string)
		scope.SetAttributes(map[string]any{
			"rbac.user_role": role,
		})

		for _, requiredRole := range requiredRoles {
			if role == requiredRole {
				scope.SetAttributes(map[string]any{
					"rbac.result":       "success",
					"rbac.matched_role": requiredRole,
				})
				return c.Next()
			}
		}

		scope.SetAttributes(map[string]any{
			"rbac.result": "failed",
			"rbac.reason": "insufficient_permissions",
		})

		log.Warn().
			Str("user_role", role).
			Strs("required_roles", requiredRoles).
			Str("path", c.Path()).
			Msg("Insufficient permissions")

		return response.WithError(c, failure.Forbidden("Insufficient permissions"))
	}
}

// RequireSuperAdmin is a convenience function for super admin only access
func (m *authRoleImpl) RequireSuperAdmin() fiber.Handler {
	return m.RBAC(constant.RoleSuperAdmin)
}

// RequireAdmin is a convenience function for admin or super admin access
func (m *authRoleImpl) RequireAdmin() fiber.Handler {
	return m.RBAC(constant.RoleSuperAdmin, constant.RoleAdmin)
}

// RequireUser is a convenience function for any authenticated user access
func (m *authRoleImpl) RequireUser() fiber.Handler {
	return m.RBAC(constant.RoleSuperAdmin, constant.RoleAdmin, constant.RoleUser)
}
