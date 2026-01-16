package service

import (
	"context"
	"fmt"
	"strings"

	"oil/config"
	"oil/infras/otel"
	"oil/infras/s3"
	"oil/internal/domains/room/model"
	"oil/internal/domains/room/model/dto"
	"oil/internal/domains/room/repository"
	"oil/shared"
	"oil/shared/cache"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/failure"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	cacheGetRoom    = "room:get"
	cacheGetAllRoom = "room:gets"
	cacheCountRoom  = "room:count"
)

type Room interface {
	Create(ctx context.Context, req dto.CreateRoomRequest) error
	GetAll(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (dto.GetRoomsResponse, error)
	Count(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (int, error)
	Get(ctx context.Context, id string) (dto.RoomResponse, error)
	Update(ctx context.Context, req dto.UpdateRoomRequest, id string) error
	Delete(ctx context.Context, id string) error
}

type serviceImpl struct {
	repo  repository.Room
	cfg   *config.Config
	cache cache.RedisCache
	otel  otel.Otel
	s3    s3.S3
}

func New(repo repository.Room, cfg *config.Config, cache cache.RedisCache, otel otel.Otel, s3 s3.S3) Room {
	return &serviceImpl{
		repo:  repo,
		cfg:   cfg,
		cache: cache,
		otel:  otel,
		s3:    s3,
	}
}

func (s *serviceImpl) Create(ctx context.Context, req dto.CreateRoomRequest) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Create")
	defer scope.End()
	defer scope.TraceIfError(err)

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)

	imageURL := constant.Empty
	var uploadedObjectName string
	if req.Image != nil {
		bucketName := s.cfg.External.S3.BucketName
		filename := uuid.NewString()

		// Get original extension
		parts := strings.Split(req.Image.Filename, ".")
		if len(parts) > 1 {
			filename = fmt.Sprintf("%s.%s", filename, parts[len(parts)-1])
		}

		url, err := s.s3.UploadFile(ctx, bucketName, model.EntityName, req.ImageFile, req.Image, filename)
		if err != nil {
			log.Error().Err(err).Msg("failed to upload image to S3")

			return fmt.Errorf("failed to upload image: %w", err)
		}
		imageURL = url
		uploadedObjectName = filename
	}

	if err = s.repo.Insert(ctx, req.ToModel(user, imageURL)); err != nil {
		if uploadedObjectName != constant.Empty {
			bucketName := s.cfg.External.S3.BucketName
			_ = s.s3.DeleteFile(ctx, bucketName, model.EntityName, uploadedObjectName)
		}

		return err
	}

	go func() {
		c := context.WithoutCancel(ctx)

		shared.InvalidateCaches(c, s.cache, cacheGetAllRoom)
		shared.InvalidateCaches(c, s.cache, cacheCountRoom)
	}()

	return nil
}

func (s *serviceImpl) GetAll(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (res dto.GetRoomsResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".GetAll")
	defer scope.End()
	defer scope.TraceIfError(err)

	cacheKey := shared.BuildCacheKeyWithQuery(cacheGetAllRoom, req, filter)

	err = s.cache.Get(ctx, cacheKey, &res)
	if err == nil {
		log.Info().Str("cacheKey", cacheKey).Msg("cache hit for rooms")

		return res, nil
	}

	total, err := s.Count(ctx, req, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to count rooms")

		return res, fmt.Errorf("failed to count rooms: %w", err)
	}

	models, err := s.repo.GetAll(ctx, req, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to get rooms")

		return res, fmt.Errorf("failed to get rooms: %w", err)
	}

	res.FromModels(models, total, req.Limit)

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Save(c, cacheKey, res, s.cfg.Cache.TTL); err != nil {
			log.Error().Err(err).Msg("failed to save rooms to cache")
		}
	}()

	return res, nil
}

func (s *serviceImpl) Count(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (res int, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Count")
	defer scope.End()
	defer scope.TraceIfError(err)

	cacheKey := shared.BuildCacheKeyWithQuery(cacheCountRoom, req, filter)

	err = s.cache.Get(ctx, cacheKey, &res)
	if err == nil {
		log.Info().Str("cacheKey", cacheKey).Msg("cache hit for room count")

		return res, nil
	}

	res, err = s.repo.Count(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to count rooms")

		return res, fmt.Errorf("failed to count rooms: %w", err)
	}

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Save(c, cacheKey, res, s.cfg.Cache.TTL); err != nil {
			log.Error().Err(err).Msg("failed to save room count to cache")
		}
	}()

	return res, nil
}

func (s *serviceImpl) Get(ctx context.Context, id string) (res dto.RoomResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Get")
	defer scope.End()
	defer scope.TraceIfError(nil)

	cacheKey := shared.BuildCacheKey(cacheGetRoom, id)

	err = s.cache.Get(ctx, cacheKey, &res)
	if err == nil {
		log.Info().Str("cacheKey", cacheKey).Msg("cache hit for room")

		return res, nil
	}

	room, err := s.repo.Get(ctx, shared.FilterByID(id, model.FieldID, model.TableName))
	if err != nil {
		log.Error().Err(err).Msg("failed to get room")

		return res, fmt.Errorf("failed to get room: %w", err)
	}

	if room.ID == constant.Empty {
		return res, failure.NotFound("room not found") // nolint:wrapcheck
	}

	res.FromModel(room)

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Save(c, cacheKey, res, s.cfg.Cache.TTL); err != nil {
			log.Error().Err(err).Msg("failed to save room to cache")
		}
	}()

	return res, nil
}

func (s *serviceImpl) Update(ctx context.Context, req dto.UpdateRoomRequest, id string) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Update")
	defer scope.End()
	defer scope.TraceIfError(err)

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)
	filter := shared.FilterByID(id, model.FieldID, model.TableName)

	currentRoom, err := s.repo.Get(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to check room existence")

		return err
	}

	if currentRoom.ID == constant.Empty {
		log.Error().Msg("room not found")

		return failure.NotFound("room not found")
	}

	return s.updateInternal(ctx, req, currentRoom, user, filter)
}

func (s *serviceImpl) updateInternal(ctx context.Context, req dto.UpdateRoomRequest, currentRoom model.Room, user string, filter gDto.FilterGroup) error {
	imageURL := constant.Empty
	var uploadedObjectName string
	bucketName := s.cfg.External.S3.BucketName

	if req.Image != nil {
		filename := uuid.NewString()

		// Get original extension
		parts := strings.Split(req.Image.Filename, ".")
		if len(parts) > 1 {
			filename = fmt.Sprintf("%s.%s", filename, parts[len(parts)-1])
		}

		url, err := s.s3.UploadFile(ctx, bucketName, model.EntityName, req.ImageFile, req.Image, filename)
		if err != nil {
			return fmt.Errorf("failed to upload image: %w", err)
		}
		imageURL = url
		uploadedObjectName = filename
	}

	updatedFields := shared.TransformFields(req, user)
	if imageURL != constant.Empty {
		updatedFields[model.FieldImage] = imageURL
	}

	if err := s.repo.Update(ctx, updatedFields, filter); err != nil {
		log.Error().Err(err).Msg("failed to update room")

		// Cleanup: delete newly uploaded image if DB update fails
		if uploadedObjectName != constant.Empty {
			_ = s.s3.DeleteFile(ctx, bucketName, model.EntityName, uploadedObjectName)
		}

		return fmt.Errorf("failed to update room: %w", err)
	}

	// Delete old image if update succeeded and new image was uploaded
	if imageURL != constant.Empty && currentRoom.Image != constant.Empty {
		oldObjectName := s.s3.GetObjectNameFromURL(bucketName, currentRoom.Image)
		if oldObjectName != constant.Empty {
			_ = s.s3.DeleteFile(ctx, bucketName, model.EntityName, oldObjectName)
		}
	}

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Delete(c, shared.BuildCacheKey(cacheGetRoom, currentRoom.ID)); err != nil {
			log.Error().Err(err).Msg("failed to delete room cache")
		}

		shared.InvalidateCaches(c, s.cache, cacheGetAllRoom)
		shared.InvalidateCaches(c, s.cache, cacheCountRoom)
	}()

	return nil
}

func (s *serviceImpl) Delete(ctx context.Context, id string) error {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Delete")
	defer scope.End()
	defer scope.TraceIfError(nil)

	exist, err := s.repo.Exist(ctx, shared.FilterByID(id, model.FieldID, model.TableName))
	if err != nil {
		log.Error().Err(err).Msg("failed to check if room exists")

		return fmt.Errorf("failed to check if room exists: %w", err)
	}

	if !exist {
		log.Error().Msg("room not found")

		return failure.NotFound("room not found") // nolint:wrapcheck
	}

	if err := s.repo.Delete(ctx, shared.FilterByID(id, model.FieldID, model.TableName)); err != nil {
		log.Error().Err(err).Msg("failed to delete room")

		return fmt.Errorf("failed to delete room: %w", err)
	}

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Delete(c, shared.BuildCacheKey(cacheGetRoom, id)); err != nil {
			log.Error().Err(err).Msg("failed to delete room from cache")
		}

		shared.InvalidateCaches(c, s.cache, cacheGetAllRoom)
		shared.InvalidateCaches(c, s.cache, cacheCountRoom)
	}()

	return nil
}
