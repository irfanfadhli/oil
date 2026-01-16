package service

import (
	"context"
	"fmt"
	"oil/config"
	"oil/infras/otel"
	"oil/internal/domains/booking/model"
	"oil/internal/domains/booking/model/dto"
	"oil/internal/domains/booking/repository"
	roomModel "oil/internal/domains/room/model"
	roomRepo "oil/internal/domains/room/repository"
	"oil/shared"
	"oil/shared/cache"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/failure"

	"github.com/rs/zerolog/log"
)

const (
	cacheGetBooking    = "booking:get"
	cacheGetAllBooking = "booking:gets"
	cacheCountBooking  = "booking:count"
)

type Booking interface {
	Create(ctx context.Context, req dto.CreateBookingRequest) error
	GetAll(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (dto.GetBookingsResponse, error)
	Count(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (int, error)
	Get(ctx context.Context, id string) (dto.BookingResponse, error)
	Update(ctx context.Context, req dto.UpdateBookingRequest, id string) error
	Delete(ctx context.Context, id string) error
}

type serviceImpl struct {
	repo     repository.Booking
	roomRepo roomRepo.Room
	cfg      *config.Config
	cache    cache.RedisCache
	otel     otel.Otel
}

func New(repo repository.Booking, roomRepo roomRepo.Room, cfg *config.Config, cache cache.RedisCache, otel otel.Otel) Booking {
	return &serviceImpl{
		repo:     repo,
		roomRepo: roomRepo,
		cfg:      cfg,
		cache:    cache,
		otel:     otel,
	}
}

func (s *serviceImpl) Create(ctx context.Context, req dto.CreateBookingRequest) (err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Create")
	defer scope.End()
	defer scope.TraceIfError(err)

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)

	// Validate that the room exists
	roomExists, err := s.roomRepo.Exist(ctx, shared.FilterByID(req.RoomID, roomModel.FieldID, roomModel.TableName))
	if err != nil {
		log.Error().Err(err).Msg("failed to check if room exists")

		return fmt.Errorf("failed to check if room exists: %w", err)
	}

	if !roomExists {
		return failure.BadRequestFromString("room does not exist") // nolint:wrapcheck
	}

	booking, err := req.ToModel(user)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse booking request")

		return failure.BadRequestFromString(fmt.Sprintf("invalid date/time format: %v", err)) // nolint:wrapcheck
	}

	if err = s.repo.Insert(ctx, booking); err != nil {
		log.Error().Err(err).Msg("failed to create booking")

		return fmt.Errorf("failed to create booking: %w", err)
	}

	go func() {
		c := context.WithoutCancel(ctx)

		shared.InvalidateCaches(c, s.cache, cacheGetAllBooking)
		shared.InvalidateCaches(c, s.cache, cacheCountBooking)
	}()

	return nil
}

func (s *serviceImpl) GetAll(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (res dto.GetBookingsResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".GetAll")
	defer scope.End()
	defer scope.TraceIfError(err)

	cacheKey := shared.BuildCacheKeyWithQuery(cacheGetAllBooking, req, filter)

	err = s.cache.Get(ctx, cacheKey, &res)
	if err == nil {
		log.Info().Str("cacheKey", cacheKey).Msg("cache hit for bookings")

		return res, nil
	}

	total, err := s.Count(ctx, req, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to count bookings")

		return res, fmt.Errorf("failed to count bookings: %w", err)
	}

	models, err := s.repo.GetAll(ctx, req, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to get bookings")

		return res, fmt.Errorf("failed to get bookings: %w", err)
	}

	res.FromModels(models, total, req.Limit)

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Save(c, cacheKey, res, s.cfg.Cache.TTL); err != nil {
			log.Error().Err(err).Msg("failed to save bookings to cache")
		}
	}()

	return res, nil
}

func (s *serviceImpl) Count(ctx context.Context, req gDto.QueryParams, filter gDto.FilterGroup) (res int, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Count")
	defer scope.End()
	defer scope.TraceIfError(err)

	cacheKey := shared.BuildCacheKeyWithQuery(cacheCountBooking, req, filter)

	err = s.cache.Get(ctx, cacheKey, &res)
	if err == nil {
		log.Info().Str("cacheKey", cacheKey).Msg("cache hit for booking count")

		return res, nil
	}

	res, err = s.repo.Count(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to count bookings")

		return res, fmt.Errorf("failed to count bookings: %w", err)
	}

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Save(c, cacheKey, res, s.cfg.Cache.TTL); err != nil {
			log.Error().Err(err).Msg("failed to save booking count to cache")
		}
	}()

	return res, nil
}

func (s *serviceImpl) Get(ctx context.Context, id string) (res dto.BookingResponse, err error) {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Get")
	defer scope.End()
	defer scope.TraceIfError(nil)

	cacheKey := shared.BuildCacheKey(cacheGetBooking, id)

	err = s.cache.Get(ctx, cacheKey, &res)
	if err == nil {
		log.Info().Str("cacheKey", cacheKey).Msg("cache hit for booking")

		return res, nil
	}

	booking, err := s.repo.Get(ctx, shared.FilterByID(id, model.FieldID, model.TableName))
	if err != nil {
		log.Error().Err(err).Msg("failed to get booking")

		return res, fmt.Errorf("failed to get booking: %w", err)
	}

	if booking.ID == constant.Empty {
		return res, failure.NotFound("booking not found") // nolint:wrapcheck
	}

	res.FromModel(booking)

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Save(c, cacheKey, res, s.cfg.Cache.TTL); err != nil {
			log.Error().Err(err).Msg("failed to save booking to cache")
		}
	}()

	return res, nil
}

func (s *serviceImpl) Update(ctx context.Context, req dto.UpdateBookingRequest, id string) error {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Update")
	defer scope.End()
	defer scope.TraceIfError(nil)

	if req == (dto.UpdateBookingRequest{}) {
		return failure.BadRequestFromString("update request cannot be empty") // nolint:wrapcheck
	}

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)
	filter := shared.FilterByID(id, model.FieldID, model.TableName)

	exist, err := s.repo.Exist(ctx, filter)
	if err != nil {
		log.Error().Err(err).Msg("failed to check if booking exists")

		return fmt.Errorf("failed to check if booking exists: %w", err)
	}

	if !exist {
		log.Error().Msg("booking not found")

		return failure.NotFound("booking not found") // nolint:wrapcheck
	}

	updatedFields := shared.TransformFields(req, user)
	if err := s.repo.Update(ctx, updatedFields, filter); err != nil {
		log.Error().Err(err).Msg("failed to update booking")

		return fmt.Errorf("failed to update booking: %w", err)
	}

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Delete(c, shared.BuildCacheKey(cacheGetBooking, id)); err != nil {
			log.Error().Err(err).Msg("failed to delete booking from cache")
		}

		shared.InvalidateCaches(c, s.cache, cacheGetAllBooking)
		shared.InvalidateCaches(c, s.cache, cacheCountBooking)
	}()

	return nil
}

func (s *serviceImpl) Delete(ctx context.Context, id string) error {
	ctx, scope := s.otel.NewScope(ctx, constant.OtelServiceScopeName, constant.OtelServiceScopeName+".Delete")
	defer scope.End()
	defer scope.TraceIfError(nil)

	exist, err := s.repo.Exist(ctx, shared.FilterByID(id, model.FieldID, model.TableName))
	if err != nil {
		log.Error().Err(err).Msg("failed to check if booking exists")

		return fmt.Errorf("failed to check if booking exists: %w", err)
	}

	if !exist {
		log.Error().Msg("booking not found")

		return failure.NotFound("booking not found") // nolint:wrapcheck
	}

	if err := s.repo.Delete(ctx, shared.FilterByID(id, model.FieldID, model.TableName)); err != nil {
		log.Error().Err(err).Msg("failed to delete booking")

		return fmt.Errorf("failed to delete booking: %w", err)
	}

	go func() {
		c := context.WithoutCancel(ctx)

		if err := s.cache.Delete(c, shared.BuildCacheKey(cacheGetBooking, id)); err != nil {
			log.Error().Err(err).Msg("failed to delete booking from cache")
		}

		shared.InvalidateCaches(c, s.cache, cacheGetAllBooking)
		shared.InvalidateCaches(c, s.cache, cacheCountBooking)
	}()

	return nil
}
