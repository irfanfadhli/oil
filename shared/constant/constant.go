package constant

import (
	"time"
)

const (
	ContextGuest = "guest"
)

// Context key types to avoid collisions
type contextKey string

const (
	ContextKeyUserID    contextKey = "user_id"
	ContextKeyUserEmail contextKey = "user_email"
	ContextKeyUserRole  contextKey = "user_role"
	ContextKeyTokenID   contextKey = "token_id"
)

const (
	RoleSuperAdmin = "0"
	RoleAdmin      = "1"
	RoleUser       = "2"
)

const (
	RequestParamPage    = "page"
	RequestParamLimit   = "limit"
	RequestParamSortBy  = "sort_by"
	RequestParamSortDir = "sort_dir"
)

const (
	DefaultValuePage    = 1
	DefaultValueLimit   = 10
	DefaultValueSortBy  = "created_at"
	DefaultValueSortDir = "DESC"
)

const (
	FieldCreatedAt  = "created_at"
	FieldCreatedBy  = "created_by"
	FieldModifiedAt = "modified_at"
	FieldModifiedBy = "modified_by"
)

const (
	PqErrorCodeUniqueViolation = "23505"
	PqErrorCodeFkViolation     = "23503"
)

const (
	DateFormat = time.RFC3339
)

const (
	MinutesToSeconds = 60
)

const (
	OtelServiceScopeName    = "service"
	OtelRepositoryScopeName = "repository"
	OtelHandlerScopeName    = "handler"
	OtelEventScopeName      = "event"
	OtelExternalScopeName   = "external"

	OtelQueryAttributeKey = "query"
	OtelMinioScopeName    = "minio"
)

const (
	ContextKeyRateLimit          = "X-RateLimit-Limit"
	ContextKeyRateLimitRemaining = "X-RateLimit-Remaining"
	ContextKeyRateLimitWindow    = "X-RateLimit-Window"
	ContextKeyUserAgent          = "User-Agent"
)

const (
	ResponseErrorPrepareShutdown      = "SERVER PREPARING TO SHUT DOWN"
	ResponseErrorUnhealthy            = "SERVER UNHEALTHY"
	ResponseErrorRequestLimitExceeded = "REQUEST LIMIT EXCEEDED"
)

const (
	ServerEnvDevelopment = "development"
	ServerEnvProduction  = "production"
)
