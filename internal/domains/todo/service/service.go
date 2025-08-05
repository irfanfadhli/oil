package service

import (
	"context"
	"fmt"
	"oil/config"
	"oil/infras/otel"
	"oil/internal/domains/todo/model"
	"oil/internal/domains/todo/model/dto"
	"oil/internal/domains/todo/repository"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/failure"

	"github.com/rs/zerolog/log"
)

type Todo interface {
	Create(ctx context.Context, req dto.CreateTodoRequest) error
	GetAll(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (dto.GetTodosResponse, error)
	Get(ctx context.Context, id string) (dto.TodoResponse, error)
	Update(ctx context.Context, req dto.UpdateTodoRequest, id string) error
	Delete(ctx context.Context, id string) error
}

type serviceImpl struct {
	repo repository.Todo
	cfg  *config.Config
	otel otel.Otel
}

func New(repo repository.Todo, cfg *config.Config, otel otel.Otel) Todo {
	return &serviceImpl{
		repo: repo,
		cfg:  cfg,
		otel: otel,
	}
}

func (s *serviceImpl) Create(ctx context.Context, req dto.CreateTodoRequest) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Create")
	defer scope.End()
	defer scope.TraceIfError(err)

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)

	if err = s.repo.Insert(ctx, req.ToModel(user)); err != nil {
		log.Error().Err(err).Msg("failed to create todo")

		return fmt.Errorf("failed to create todo: %w", err)
	}

	return nil
}

func (s *serviceImpl) GetAll(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (res dto.GetTodosResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".GetAll")
	defer scope.End()
	defer scope.TraceIfError(err)

	total, err := s.repo.Count(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to count todos")

		return res, fmt.Errorf("failed to count todos: %w", err)
	}

	models, err := s.repo.GetAll(ctx, req, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to get todos")

		return res, fmt.Errorf("failed to get todos: %w", err)
	}

	res.FromModels(models, total, req.Limit)

	return res, nil
}

func (s *serviceImpl) Get(ctx context.Context, id string) (res dto.TodoResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Get")
	defer scope.End()
	defer scope.TraceIfError(nil)

	todo, err := s.repo.Get(ctx, shared.FilterByID(id, model.FieldID, model.TableName))
	if err != nil {
		log.Error().Err(err).Msg("failed to get todo")

		return res, fmt.Errorf("failed to get todo: %w", err)
	}

	if todo.ID == "" {
		return res, failure.NotFound("todo not found") // nolint:wrapcheck
	}

	res.FromModel(todo)

	return res, nil
}

func (s *serviceImpl) Update(ctx context.Context, req dto.UpdateTodoRequest, id string) error {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Update")
	defer scope.End()
	defer scope.TraceIfError(nil)

	if req == (dto.UpdateTodoRequest{}) {
		return failure.BadRequestFromString("update request cannot be empty") // nolint:wrapcheck
	}

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)
	filter := shared.FilterByID(id, model.FieldID, model.TableName)

	exist, err := s.repo.Exist(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if todo exists")

		return fmt.Errorf("failed to check if todo exists: %w", err)
	}

	if !exist {
		log.Error().Msg("todo not found")

		return failure.NotFound("todo not found") // nolint:wrapcheck
	}

	updatedFields := shared.TransformFields(req, user)
	if err := s.repo.Update(ctx, updatedFields, filter); err != nil {
		log.Error().Err(err).Msg("failed to update todo")

		return fmt.Errorf("failed to update todo: %w", err)
	}

	return nil
}

func (s *serviceImpl) Delete(ctx context.Context, id string) error {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Delete")
	defer scope.End()
	defer scope.TraceIfError(nil)

	exist, err := s.repo.Exist(ctx, shared.FilterByID(id, model.FieldID, model.TableName))
	if err != nil {
		log.Error().Err(err).Msg("failed to check if todo exists")

		return fmt.Errorf("failed to check if todo exists: %w", err)
	}

	if !exist {
		log.Error().Msg("todo not found")

		return failure.NotFound("todo not found") // nolint:wrapcheck
	}

	if err := s.repo.Delete(ctx, shared.FilterByID(id, model.FieldID, model.TableName)); err != nil {
		log.Error().Err(err).Msg("failed to delete todo")

		return fmt.Errorf("failed to delete todo: %w", err)
	}

	return nil
}
