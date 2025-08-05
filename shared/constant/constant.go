package constant

import (
	"time"
)

const (
	ContextGuest = "guest"
)

const (
	ContextKeyUserID    = "user_id"
	ContextKeyUserEmail = "user_email"
	ContextKeyUserRole  = "user_role"
	ContextKeyTokenID   = "token_id"
)

const (
	RoleSuperAdmin = "0"
	RoleAdmin      = "1"
	RoleUser       = "2"
)

const (
	ContextKeyUserAgent = "User-Agent"
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
	OtelServiceScopeName    = "service"
	OtelRepositoryScopeName = "repository"
	OtelHandlerScopeName    = "handler"
	OtelEventScopeName      = "event"
	OtelExternalScopeName   = "external"

	OtelQueryAttributeKey = "query"
	OtelMinioScopeName    = "minio"
)

const (
	ResponseErrorPrepareShutdown = "SERVER PREPARING TO SHUT DOWN"
	ResponseErrorUnhealthy       = "SERVER UNHEALTHY"
)

const (
	ServerEnvDevelopment = "development"
	ServerEnvProduction  = "production"
)
