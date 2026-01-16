package service

import (
	"context"
	"fmt"
	"oil/config"
	"oil/infras/otel"
	"oil/internal/domains/user/model"
	"oil/internal/domains/user/model/dto"
	"oil/internal/domains/user/repository"
	"oil/shared"
	"oil/shared/cache"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/failure"
	"oil/shared/password"

	"github.com/rs/zerolog/log"
)

const (
	cacheGetUser    = "user:get"
	cacheGetAllUser = "user:gets"
	cacheCountUser  = "user:count"
)

type User interface {
	Create(ctx context.Context, req dto.CreateUserRequest) error
	GetAll(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (dto.GetUsersResponse, error)
	Count(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (int, error)
	Get(ctx context.Context, id string) (dto.UserResponse, error)
	Update(ctx context.Context, req dto.UpdateUserRequest, id string) error
	Delete(ctx context.Context, id string) error
}

type serviceImpl struct {
	repo  repository.User
	cfg   *config.Config
	cache cache.RedisCache
	otel  otel.Otel
}

func New(repo repository.User, cfg *config.Config, cache cache.RedisCache, otel otel.Otel) User {
	return &serviceImpl{
		repo:  repo,
		cfg:   cfg,
		cache: cache,
		otel:  otel,
	}
}

func (s *serviceImpl) Create(ctx context.Context, req dto.CreateUserRequest) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Create")
	defer scope.End()
	defer scope.TraceIfError(err)

	user, _ := ctx.Value(constant.ContextGuest).(string)

	emailFilter := gDto.FilterGroup{
		Filters: []any{
			gDto.Filter{
				Field:    model.FieldEmail,
				Operator: gDto.FilterOperatorEq,
				Value:    req.Email,
				Table:    model.TableName,
			},
		},
	}

	exists, err := s.repo.Exist(ctx, emailFilter)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if user exists")

		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	if exists {
		return failure.BadRequestFromString("email already registered")
	}

	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		log.Error().Err(err).Msg("failed to hash password")

		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err = s.repo.Insert(ctx, req.ToModel(user, hashedPassword)); err != nil {
		log.Error().Err(err).Msg("failed to create user")

		return fmt.Errorf("failed to create user: %w", err)
	}

	go func() {
		c := context.WithoutCancel(ctx)

		shared.InvalidateCaches(c, s.cache, cacheGetAllUser)
		shared.InvalidateCaches(c, s.cache, cacheCountUser)
	}()

	return nil
}

func (s *serviceImpl) GetAll(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (res dto.GetUsersResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".GetAll")
	defer scope.End()
	defer scope.TraceIfError(err)

	cacheKey := shared.BuildCacheKeyWithQuery(cacheGetAllUser, req, filter)

	err = s.cache.Get(ctx, cacheKey, &res)
	if err == nil {
		log.Info().Str("cacheKey", cacheKey).Msg("cache hit for users")

		return res, nil
	}

	total, err := s.Count(ctx, req, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to count users")

		return res, fmt.Errorf("failed to count users: %w", err)
	}

	models, err := s.repo.GetAll(ctx, req, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to get users")

		return res, fmt.Errorf("failed to get users: %w", err)
	}

	res.FromModels(models, total, req.Limit)

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Save(c, cacheKey, res, s.cfg.Cache.TTL); err != nil {
			log.Error().Err(err).Msg("failed to save users to cache")
		}
	}()

	return res, nil
}

func (s *serviceImpl) Count(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (res int, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Count")
	defer scope.End()
	defer scope.TraceIfError(err)

	cacheKey := shared.BuildCacheKeyWithQuery(cacheCountUser, req, filter)

	err = s.cache.Get(ctx, cacheKey, &res)
	if err == nil {
		log.Info().Str("cacheKey", cacheKey).Msg("cache hit for user count")

		return res, nil
	}

	res, err = s.repo.Count(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to count users")

		return res, fmt.Errorf("failed to count users: %w", err)
	}

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Save(c, cacheKey, res, s.cfg.Cache.TTL); err != nil {
			log.Error().Err(err).Msg("failed to save user count to cache")
		}
	}()

	return res, nil
}

func (s *serviceImpl) Get(ctx context.Context, id string) (res dto.UserResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Get")
	defer scope.End()
	defer scope.TraceIfError(nil)

	cacheKey := shared.BuildCacheKey(cacheGetUser, id)

	err = s.cache.Get(ctx, cacheKey, &res)
	if err == nil {
		log.Info().Str("cacheKey", cacheKey).Msg("cache hit for user")

		return res, nil
	}

	user, err := s.repo.Get(ctx, shared.FilterByID(id, model.FieldID, model.TableName))
	if err != nil {
		log.Error().Err(err).Msg("failed to get user")

		return res, fmt.Errorf("failed to get user: %w", err)
	}

	if user.ID == "" {
		return res, failure.NotFound("user not found")
	}

	res.FromModel(user)

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Save(c, cacheKey, res, s.cfg.Cache.TTL); err != nil {
			log.Error().Err(err).Msg("failed to save user to cache")
		}
	}()

	return res, nil
}

func (s *serviceImpl) Update(ctx context.Context, req dto.UpdateUserRequest, id string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Update")
	defer scope.End()
	defer scope.TraceIfError(nil)

	if req == (dto.UpdateUserRequest{}) {
		return failure.BadRequestFromString("update request cannot be empty")
	}

	user, _ := ctx.Value(constant.ContextGuest).(string)
	filter := shared.FilterByID(id, model.FieldID, model.TableName)

	exist, err := s.repo.Exist(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if user exists")

		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	if !exist {
		log.Error().Msg("user not found")

		return failure.NotFound("user not found")
	}

	updatedFields := shared.TransformFields(req, user)
	if err := s.repo.Update(ctx, updatedFields, filter); err != nil {
		log.Error().Err(err).Msg("failed to update user")

		return fmt.Errorf("failed to update user: %w", err)
	}

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Delete(c, shared.BuildCacheKey(cacheGetUser, id)); err != nil {
			log.Error().Err(err).Msg("failed to delete user from cache")
		}

		shared.InvalidateCaches(c, s.cache, cacheGetAllUser)
		shared.InvalidateCaches(c, s.cache, cacheCountUser)
	}()

	return nil
}

func (s *serviceImpl) Delete(ctx context.Context, id string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Delete")
	defer scope.End()
	defer scope.TraceIfError(nil)

	exist, err := s.repo.Exist(ctx, shared.FilterByID(id, model.FieldID, model.TableName))
	if err != nil {
		log.Error().Err(err).Msg("failed to check if user exists")

		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	if !exist {
		log.Error().Msg("user not found")

		return failure.NotFound("user not found")
	}

	if err := s.repo.Delete(ctx, shared.FilterByID(id, model.FieldID, model.TableName)); err != nil {
		log.Error().Err(err).Msg("failed to delete user")

		return fmt.Errorf("failed to delete user: %w", err)
	}

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Delete(c, shared.BuildCacheKey(cacheGetUser, id)); err != nil {
			log.Error().Err(err).Msg("failed to delete user from cache")
		}

		shared.InvalidateCaches(c, s.cache, cacheGetAllUser)
		shared.InvalidateCaches(c, s.cache, cacheCountUser)
	}()

	return nil
}
