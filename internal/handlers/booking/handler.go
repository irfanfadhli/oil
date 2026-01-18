package booking

import (
	"net/http"
	"oil/infras/otel"
	"oil/internal/domains/booking/model"
	"oil/internal/domains/booking/model/dto"
	"oil/internal/domains/booking/service"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/failure"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"

	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.Booking
	otel    otel.Otel
}

func New(service service.Booking, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(router chi.Router) {
	router.Route("/bookings", func(routerGroup chi.Router) {
		routerGroup.Post("/", handler.CreateBooking)
		routerGroup.Get("/", handler.GetBookings)
		routerGroup.Get("/mybookings", handler.GetMyBookings)
		routerGroup.Get("/{id}", handler.GetBookingByID)
		routerGroup.Patch("/{id}", handler.UpdateBooking)
		routerGroup.Delete("/{id}", handler.DeleteBooking)
	})
}

// CreateBooking handles the creation of a new booking.
// @Summary Create a new booking
// @Description Create a new room booking with the provided details.
// @Tags Booking
// @Accept json
// @Produce json
// @Param request body dto.CreateBookingRequest true "Create Booking Request"
// @Success 201 {object} response.Message "Booking created successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/bookings [post]
// @Security BearerAuth
func (handler *Handler) CreateBooking(writer http.ResponseWriter, request *http.Request) {
	ctx, scope := handler.otel.NewScope(request.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".CreateBooking")
	defer scope.End()

	req := dto.CreateBookingRequest{}

	if err := validator.Validate(request.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(writer, err)

		return
	}

	if err := handler.service.Create(ctx, req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to create booking")

		response.WithError(writer, err)

		return
	}

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)
	scope.AddEvent("Booking created successfully by user " + user)

	response.WithMessage(writer, http.StatusCreated, "Booking created successfully")
}

// GetBookings retrieves all bookings based on query parameters.
// @Summary Get all bookings
// @Description Retrieve all bookings with optional filtering and pagination.
// @Tags Booking
// @Accept json
// @Produce json
// @Param pagination query gDto.QueryParams false "Pagination parameters"
// @Param room_id query string false "Filter by room ID"
// @Param status query string false "Filter by status (pending, confirmed, cancelled)"
// @Param booking_date query string false "Filter by booking date (YYYY-MM-DD)"
// @Success 200 {object} response.Data[dto.BookingResponse] "List of bookings"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/bookings [get]
func (handler *Handler) GetBookings(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetBookings")
	defer scope.End()

	queryParams := gDto.QueryParams{}
	queryParams.FromRequest(r, true)

	roomID := r.URL.Query().Get(model.FieldRoomID)
	status := r.URL.Query().Get(model.FieldStatus)
	bookingDate := r.URL.Query().Get(model.FieldBookingDate)

	filterGroup := gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters:  []any{},
	}

	// Only add filters if the values are non-empty
	if roomID != "" {
		filterGroup.Filters = append(filterGroup.Filters, gDto.Filter{
			Field:    model.FieldRoomID,
			Operator: gDto.FilterOperatorEq,
			Value:    roomID,
			Table:    model.TableName,
		})
	}

	if status != "" {
		filterGroup.Filters = append(filterGroup.Filters, gDto.Filter{
			Field:    model.FieldStatus,
			Operator: gDto.FilterOperatorEq,
			Value:    status,
			Table:    model.TableName,
		})
	}

	if bookingDate != "" {
		filterGroup.Filters = append(filterGroup.Filters, gDto.Filter{
			Field:    model.FieldBookingDate,
			Operator: gDto.FilterOperatorEq,
			Value:    bookingDate,
			Table:    model.TableName,
		})
	}

	bookings, err := handler.service.GetAll(ctx, queryParams, filterGroup)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get bookings")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("Bookings retrieved successfully")

	response.WithJSON(w, http.StatusOK, bookings)
}

// GetMyBookings retrieves all bookings for the currently authenticated user.
// @Summary Get my bookings
// @Description Retrieve all bookings for the currently authenticated user with optional filtering and pagination.
// @Tags Booking
// @Accept json
// @Produce json
// @Param pagination query gDto.QueryParams false "Pagination parameters"
// @Param status query string false "Filter by status (pending, confirmed, cancelled)"
// @Param booking_date query string false "Filter by booking date (YYYY-MM-DD)"
// @Success 200 {object} response.Data[dto.BookingResponse] "List of user's bookings"
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/bookings/mybookings [get]
// @Security BearerAuth
func (handler *Handler) GetMyBookings(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetMyBookings")
	defer scope.End()

	// Get user_id from context
	userID, ok := ctx.Value(constant.ContextKeyUserID).(string)
	if !ok || userID == "" {
		scope.TraceError(nil)
		log.Error().Msg("failed to get user ID from context")
		response.WithError(w, failure.Unauthorized("unauthorized"))

		return
	}

	queryParams := gDto.QueryParams{}
	queryParams.FromRequest(r, true)

	status := r.URL.Query().Get(model.FieldStatus)
	bookingDate := r.URL.Query().Get(model.FieldBookingDate)

	filterGroup := gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters: []any{
			// Always filter by created_by (user_id)
			gDto.Filter{
				Field:    model.FieldCreatedBy,
				Operator: gDto.FilterOperatorEq,
				Value:    userID,
				Table:    model.TableName,
			},
		},
	}

	if status != "" {
		filterGroup.Filters = append(filterGroup.Filters, gDto.Filter{
			Field:    model.FieldStatus,
			Operator: gDto.FilterOperatorEq,
			Value:    status,
			Table:    model.TableName,
		})
	}

	if bookingDate != "" {
		filterGroup.Filters = append(filterGroup.Filters, gDto.Filter{
			Field:    model.FieldBookingDate,
			Operator: gDto.FilterOperatorEq,
			Value:    bookingDate,
			Table:    model.TableName,
		})
	}

	bookings, err := handler.service.GetAll(ctx, queryParams, filterGroup)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get user bookings")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("User bookings retrieved successfully for user " + userID)

	response.WithJSON(w, http.StatusOK, bookings)
}

// GetBookingByID retrieves a booking by its ID.
// @Summary Get a booking by ID
// @Description Retrieve a booking by its unique identifier.
// @Tags Booking
// @Accept json
// @Produce json
// @Param id path string true "Booking ID"
// @Success 200 {object} response.Data[dto.BookingResponse] "Booking details"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/bookings/{id} [get]
func (handler *Handler) GetBookingByID(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetBookingByID")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	booking, err := handler.service.Get(ctx, id)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get booking by ID")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("Booking retrieved successfully")

	response.WithJSON(w, http.StatusOK, booking)
}

// UpdateBooking updates an existing booking by its ID.
// @Summary Update a booking by ID
// @Description Update the details of an existing booking.
// @Tags Booking
// @Accept json
// @Produce json
// @Param id path string true "Booking ID"
// @Param request body dto.UpdateBookingRequest true "Update Booking Request"
// @Success 200 {object} response.Message "Booking updated successfully"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/bookings/{id} [patch]
// @Security BearerAuth
func (handler *Handler) UpdateBooking(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".UpdateBooking")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	req := dto.UpdateBookingRequest{}
	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(w, err)

		return
	}

	if err := handler.service.Update(ctx, req, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to update booking")

		response.WithError(w, err)

		return
	}

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)
	scope.AddEvent("Booking updated successfully by user " + user)

	response.WithMessage(w, http.StatusOK, "Booking updated successfully")
}

// DeleteBooking deletes a booking by its ID.
// @Summary Delete a booking by ID
// @Description Delete/cancel a booking using its unique identifier.
// @Tags Booking
// @Accept json
// @Produce json
// @Param id path string true "Booking ID"
// @Success 200 {object} response.Message "Booking deleted successfully"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/bookings/{id} [delete]
// @Security BearerAuth
func (handler *Handler) DeleteBooking(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".DeleteBooking")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	if err := handler.service.Delete(ctx, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to delete booking")

		response.WithError(w, err)

		return
	}

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)
	scope.AddEvent("Booking deleted successfully by user " + user)

	response.WithMessage(w, http.StatusOK, "Booking deleted successfully")
}
