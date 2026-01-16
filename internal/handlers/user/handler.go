package user

import (
	"net/http"
	"oil/infras/otel"
	"oil/internal/domains/user/model"
	"oil/internal/domains/user/model/dto"
	"oil/internal/domains/user/service"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/validator"
	"oil/transport/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service service.User
	otel    otel.Otel
}

func New(service service.User, otel otel.Otel) Handler {
	return Handler{
		service: service,
		otel:    otel,
	}
}

func (handler *Handler) Router(router chi.Router) {
	router.Route("/users", func(routerGroup chi.Router) {
		routerGroup.Post("/", handler.CreateUser)
		routerGroup.Get("/", handler.GetUsers)
		routerGroup.Get("/{id}", handler.GetUserByID)
		routerGroup.Patch("/{id}", handler.UpdateUser)
		routerGroup.Delete("/{id}", handler.DeleteUser)
	})
}

// CreateUser handles the creation of a new user.
// @Summary Create a new user
// @Description Create a new user with the provided details.
// @Tags User
// @Accept json
// @Produce json
// @Param request body dto.CreateUserRequest true "Create User Request"
// @Success 201 {object} response.Message "User created successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/users [post]
// @Security BearerAuth
func (handler *Handler) CreateUser(writer http.ResponseWriter, request *http.Request) {
	ctx, scope := handler.otel.NewScope(request.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".CreateUser")
	defer scope.End()

	req := dto.CreateUserRequest{}

	if err := validator.Validate(request.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(writer, err)

		return
	}

	if err := handler.service.Create(ctx, req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to create user")

		response.WithError(writer, err)

		return
	}

	scope.AddEvent("User created successfully")

	response.WithMessage(writer, http.StatusCreated, "User created successfully")
}

// GetUsers retrieves all users based on query parameters.
// @Summary Get all users
// @Description Retrieve all users with optional filtering and pagination.
// @Tags User
// @Accept json
// @Produce json
// @Param pagination query gDto.QueryParams false "Pagination parameters"
// @Param email query string false "Filter by email"
// @Param level query string false "Filter by level"
// @Success 200 {object} response.Data[dto.UserResponse] "List of users"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/users [get]
// @Security BearerAuth
func (handler *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetUsers")
	defer scope.End()

	queryParams := gDto.QueryParams{}
	queryParams.FromRequest(r, true)

	email := r.URL.Query().Get(model.FieldEmail)
	level := r.URL.Query().Get(model.FieldLevel)

	filterGroup := gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters: []any{
			gDto.Filter{
				Field:    model.FieldEmail,
				Operator: gDto.FilterOperatorEq,
				Value:    email,
				Table:    model.TableName,
			},
			gDto.Filter{
				Field:    model.FieldLevel,
				Operator: gDto.FilterOperatorEq,
				Value:    level,
				Table:    model.TableName,
			},
		},
	}

	users, err := handler.service.GetAll(ctx, queryParams, filterGroup)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get users")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("Users retrieved successfully")

	response.WithJSON(w, http.StatusOK, users)
}

// GetUserByID retrieves a user by their ID.
// @Summary Get a user by ID
// @Description Retrieve a user by their unique identifier.
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} response.Data[dto.UserResponse] "User details"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/users/{id} [get]
// @Security BearerAuth
func (handler *Handler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetUserByID")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	user, err := handler.service.Get(ctx, id)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get user by ID")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("User retrieved successfully")

	response.WithJSON(w, http.StatusOK, user)
}

// UpdateUser updates an existing user by their ID.
// @Summary Update a user by ID
// @Description Update the details of an existing user.
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body dto.UpdateUserRequest true "Update User Request"
// @Success 200 {object} response.Message "User updated successfully"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/users/{id} [patch]
// @Security BearerAuth
func (handler *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".UpdateUser")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	req := dto.UpdateUserRequest{}
	if err := validator.Validate(r.Body, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		response.WithError(w, err)

		return
	}

	if err := handler.service.Update(ctx, req, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to update user")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("User updated successfully")

	response.WithMessage(w, http.StatusOK, "User updated successfully")
}

// DeleteUser deletes a user by their ID.
// @Summary Delete a user by ID
// @Description Delete a user using their unique identifier.
// @Tags User
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} response.Message "User deleted successfully"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/users/{id} [delete]
// @Security BearerAuth
func (handler *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx, scope := handler.otel.NewScope(r.Context(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".DeleteUser")
	defer scope.End()

	id := chi.URLParam(r, constant.RequestParamID)

	if err := handler.service.Delete(ctx, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to delete user")

		response.WithError(w, err)

		return
	}

	scope.AddEvent("User deleted successfully")

	response.WithMessage(w, http.StatusOK, "User deleted successfully")
}
