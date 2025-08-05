package mocks

import (
	"context"
	"oil/infras/otel"
)

type otelImpl struct {
}

// NewScope implements otel.Otel.
func (o *otelImpl) NewScope(ctx context.Context, _, _ string) (context.Context, otel.Scope) {
	return ctx, NewScope()
}

func NewOtel() otel.Otel {
	return &otelImpl{}
}
