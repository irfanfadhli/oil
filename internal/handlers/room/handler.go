package room

import (
	"net/http"
	"oil/infras/otel"
	"oil/internal/domains/room/model"
	"oil/internal/domains/room/model/dto"
	"oil/internal/domains/room/service"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"

	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.Room
	otel    otel.Otel
}

func New(service service.Room, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(router chi.Router) {
	router.Route("/rooms", func(routerGroup chi.Router) {
		routerGroup.Post("/", handler.CreateRoom)
		routerGroup.Get("/", handler.GetRooms)
		routerGroup.Get("/{id}", handler.GetRoomByID)
		routerGroup.Patch("/{id}", handler.UpdateRoom)
		routerGroup.Delete("/{id}", handler.DeleteRoom)
	})
}

// CreateRoom handles the creation of a new room.
// @Summary Create a new room
// @Description Create a new room with the provided details.
// @Tags Room
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Room name"
// @Param location formData string false "Room location"
// @Param capacity formData integer false "Room capacity"
// @Param active formData boolean false "Room active status"
// @Param image formData file false "Room image"
// @Success 201 {object} response.Message "Room created successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/rooms [post]
// @Security BearerAuth
func (handler *Handler) CreateRoom(writer http.ResponseWriter, request *http.Request) {
	ctx, scope := handler.otel.NewScope(request.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".CreateRoom")
	defer scope.End()

	if err := request.ParseMultipartForm(constant.RequestMaxMemory); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to parse multipart form")
		response.WithError(writer, err)

		return
	}

	req := dto.CreateRoomRequest{
		Name:     request.FormValue("name"),
		Location: request.FormValue("location"),
	}

	if capStr := request.FormValue("capacity"); capStr != "" {
		if c, err := shared.ConvertStringToInt(capStr); err == nil {
			req.Capacity = c
		}
	}

	if activeStr := request.FormValue("active"); activeStr != "" {
		req.Active = shared.ConvertStringToBool(activeStr)
	}

	file, fileHeader, err := request.FormFile("image")
	if err == nil {
		req.Image = fileHeader
		req.ImageFile = file

		defer file.Close()
	}

	if err := validator.ValidateStruct(&req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request")

		response.WithError(writer, err)

		return
	}

	if err := handler.service.Create(ctx, req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to create room")

		response.WithError(writer, err)

		return
	}

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)
	scope.AddEvent("Room created successfully by user " + user)

	response.WithMessage(writer, http.StatusCreated, "Room created successfully")
}

// GetRooms retrieves all room items based on query parameters.
// @Summary Get all rooms
// @Description Retrieve all rooms with optional filtering and pagination.
// @Tags Room
// @Accept json
// @Produce json
// @Param pagination query gDto.QueryParams false "Pagination parameters"
// @Param name query string false "Filter by name"
// @Param location query string false "Filter by location"
// @Param active query boolean false "Filter by active status"
// @Success 200 {object} response.Data[dto.RoomResponse] "List of rooms"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/rooms [get]
func (handler *Handler) GetRooms(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetRooms")
	defer scope.End()

	queryParams := gDto.QueryParams{}
	queryParams.FromRequest(r, true)

	name := r.URL.Query().Get(model.FieldName)
	location := r.URL.Query().Get(model.FieldLocation)

	filterGroup := gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters: []any{
			gDto.Filter{
				Field:    model.FieldName,
				Operator: gDto.FilterOperatorLike,
				Value:    name,
				Table:    model.TableName,
			},
			gDto.Filter{
				Field:    model.FieldLocation,
				Operator: gDto.FilterOperatorLike,
				Value:    location,
				Table:    model.TableName,
			},
		},
	}

	if active := shared.ConvertStringToBool(r.URL.Query().Get(model.FieldActive)); active != nil {
		filterGroup.Filters = append(filterGroup.Filters, gDto.Filter{
			Field:    model.FieldActive,
			Operator: gDto.FilterOperatorEq,
			Value:    *active,
			Table:    model.TableName,
		})
	}

	rooms, err := handler.service.GetAll(ctx, queryParams, filterGroup)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get rooms")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("Rooms retrieved successfully")

	response.WithJSON(w, http.StatusOK, rooms)
}

// GetRoomByID retrieves a room by its ID.
// @Summary Get a room by ID
// @Description Retrieve a room by its unique identifier.
// @Tags Room
// @Accept json
// @Produce json
// @Param id path string true "Room ID"
// @Success 200 {object} response.Data[dto.RoomResponse] "Room details"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/rooms/{id} [get]
func (handler *Handler) GetRoomByID(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetRoomByID")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	room, err := handler.service.Get(ctx, id)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get room by ID")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("Room retrieved successfully")

	response.WithJSON(w, http.StatusOK, room)
}

// UpdateRoom updates an existing room by its ID.
// @Summary Update a room by ID
// @Description Update the details of an existing room.
// @Tags Room
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Room ID"
// @Param name formData string false "Room name"
// @Param location formData string false "Room location"
// @Param capacity formData integer false "Room capacity"
// @Param active formData boolean false "Room active status"
// @Param image formData file false "Room image"
// @Success 200 {object} response.Message "Room updated successfully"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/rooms/{id} [patch]
// @Security BearerAuth
func (handler *Handler) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".UpdateRoom")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	if err := r.ParseMultipartForm(constant.RequestMaxMemory); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to parse multipart form")
		response.WithError(w, err)

		return
	}

	req := dto.UpdateRoomRequest{
		Name:     r.FormValue("name"),
		Location: r.FormValue("location"),
	}

	if capStr := r.FormValue("capacity"); capStr != "" {
		if c, err := shared.ConvertStringToInt(capStr); err == nil {
			req.Capacity = &c
		}
	}

	if activeStr := r.FormValue("active"); activeStr != "" {
		req.Active = shared.ConvertStringToBool(activeStr)
	}

	file, fileHeader, err := r.FormFile("image")
	if err == nil {
		req.Image = fileHeader
		req.ImageFile = file

		defer file.Close()
	}

	if err := validator.ValidateStruct(&req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request")

		response.WithError(w, err)

		return
	}

	if err := handler.service.Update(ctx, req, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to update room")

		response.WithError(w, err)

		return
	}

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)
	scope.AddEvent("Room updated successfully by user " + user)

	response.WithMessage(w, http.StatusOK, "Room updated successfully")
}

// DeleteRoom deletes a room by its ID.
// @Summary Delete a room by ID
// @Description Delete a room using its unique identifier.
// @Tags Room
// @Accept json
// @Produce json
// @Param id path string true "Room ID"
// @Success 200 {object} response.Message "Room deleted successfully"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/rooms/{id} [delete]
// @Security BearerAuth
func (handler *Handler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".DeleteRoom")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	if err := handler.service.Delete(ctx, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to delete room")

		response.WithError(w, err)

		return
	}

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)
	scope.AddEvent("Room deleted successfully by user " + user)

	response.WithMessage(w, http.StatusOK, "Room deleted successfully")
}
