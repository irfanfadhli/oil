package todo

import (
	"fmt"
	"oil/infras/otel"
	"oil/internal/domains/todo/model"
	"oil/internal/domains/todo/model/dto"
	"oil/internal/domains/todo/service"
	"oil/shared"
	"oil/shared/constant"
	gDto "oil/shared/dto"
	"oil/shared/validator"
	"oil/transport/http/middleware"
	"oil/transport/http/response"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	service    service.Todo
	middleware middleware.AuthRole
	otel       otel.Otel
}

func New(service service.Todo, middleware middleware.AuthRole, otel otel.Otel) Handler {
	return Handler{
		service:    service,
		middleware: middleware,
		otel:       otel,
	}
}

func (handler *Handler) Router(r fiber.Router) {
	r.Route("/todos", func(r fiber.Router) {
		r.Post("/", handler.middleware.Auth(), handler.middleware.RequireUser(), handler.CreateTodo)
		r.Get("/", handler.GetTodos)
		r.Get("/:id", handler.GetTodoByID)
		r.Patch("/:id", handler.middleware.Auth(), handler.middleware.RequireUser(), handler.UpdateTodo)
		r.Delete("/:id", handler.middleware.Auth(), handler.middleware.RequireSuperAdmin(), handler.DeleteTodo)
	})
}

// CreateTodo handles the creation of a new todo item.
// @Summary Create a new todo item
// @Description Create a new todo item with the provided details.
// @Tags Todo
// @Accept json
// @Produce json
// @Param request body dto.CreateTodoRequest true "Create Todo Request"
// @Success 201 {object} response.Message "Todo created successfully"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/todos [post]
// @Security BearerAuth
func (handler *Handler) CreateTodo(c *fiber.Ctx) error {
	ctx, scope := handler.otel.NewScope(c.UserContext(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".CreateTodo")
	defer scope.End()

	req := dto.CreateTodoRequest{}

	if err := validator.Validate(c, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		return response.WithError(c, err)
	}

	if err := handler.service.Create(ctx, req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to create todo")

		return response.WithError(c, err)
	}

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)
	scope.AddEvent(fmt.Sprintf("Todo created successfully by user %s", user))

	return response.WithMessage(c, fiber.StatusCreated, "Todo created successfully")
}

// GetTodos retrieves all todo items based on query parameters.
// @Summary Get all todo items
// @Description Retrieve all todo items with optional filtering and pagination.
// @Tags Todo
// @Accept json
// @Produce json
// @Param title query string false "Filter by title"
// @Param completed query boolean false "Filter by completion status"
// @Success 200 {array} model.Todo "List of todo items"
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/todos [get]
func (handler *Handler) GetTodos(c *fiber.Ctx) error {
	ctx, scope := handler.otel.NewScope(c.UserContext(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetTodos")
	defer scope.End()

	queryParams := gDto.QueryParams{}
	queryParams.FromRequest(c, true)

	title := c.Query(model.FieldTitle)

	filterGroup := gDto.FilterGroup{
		Operator: gDto.FilterGroupOperatorAnd,
		Filters: []any{
			gDto.Filter{
				Field:    model.FieldTitle,
				Operator: gDto.FilterOperatorLike,
				Value:    title,
				Table:    model.TableName,
			},
		},
	}

	if complete := shared.ConvertStringToBool(c.Query(model.FieldCompleted)); complete != nil {
		filterGroup.Filters = append(filterGroup.Filters, gDto.Filter{
			Field:    model.FieldCompleted,
			Operator: gDto.FilterOperatorEq,
			Value:    *complete,
			Table:    model.TableName,
		})
	}

	todos, err := handler.service.GetAll(ctx, queryParams, filterGroup)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get todos")

		return response.WithError(c, err)
	}

	scope.AddEvent("Todos retrieved successfully")

	return response.WithJSON(c, fiber.StatusOK, todos)
}

// GetTodoByID retrieves a todo item by its ID.
// @Summary Get a todo item by ID
// @Description Retrieve a todo item by its unique identifier.
// @Tags Todo
// @Accept json
// @Produce json
// @Param id path string true "Todo ID"
// @Success 200 {object} dto.TodoResponse "Todo item details"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/todos/{id} [get]
func (handler *Handler) GetTodoByID(c *fiber.Ctx) error {
	ctx, scope := handler.otel.NewScope(c.UserContext(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".GetTodoByID")
	defer scope.End()

	id := c.Params(model.FieldID)

	todo, err := handler.service.Get(ctx, id)
	if err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to get todo by ID")

		return response.WithError(c, err)
	}

	scope.AddEvent("Todo retrieved successfully")

	return response.WithJSON(c, fiber.StatusOK, todo)
}

// UpdateTodo updates an existing todo item by its ID.
// @Summary Update a todo item by ID
// @Description Update the details of an existing todo item.
// @Tags Todo
// @Accept json
// @Produce json
// @Param id path string true "Todo ID"
// @Param request body dto.UpdateTodoRequest true "Update Todo Request"
// @Success 200 {object} response.Message "Todo updated successfully"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/todos/{id} [patch]
// @Security BearerAuth
func (handler *Handler) UpdateTodo(c *fiber.Ctx) error {
	ctx, scope := handler.otel.NewScope(c.UserContext(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".UpdateTodo")
	defer scope.End()

	id := c.Params(model.FieldID)

	req := dto.UpdateTodoRequest{}
	if err := validator.Validate(c, &req); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to validate request body")

		return response.WithError(c, err)
	}

	if err := handler.service.Update(ctx, req, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to update todo")

		return response.WithError(c, err)
	}

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)
	scope.AddEvent(fmt.Sprintf("Todo updated successfully by user %s", user))

	return response.WithMessage(c, fiber.StatusOK, "Todo updated successfully")
}

// DeleteTodo deletes a todo item by its ID.
// @Summary Delete a todo item by ID @SuperAdmin
// @Description Delete a todo item using its unique identifier.
// @Tags Todo
// @Accept json
// @Produce json
// @Param id path string true "Todo ID"
// @Success 200 {object} response.Message "Todo deleted successfully"
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /v1/todos/{id} [delete]
// @Security BearerAuth
func (handler *Handler) DeleteTodo(c *fiber.Ctx) error {
	ctx, scope := handler.otel.NewScope(c.UserContext(), constant.OtelHandlerScopeName, constant.OtelHandlerScopeName+".DeleteTodo")
	defer scope.End()

	id := c.Params(model.FieldID)

	if err := handler.service.Delete(ctx, id); err != nil {
		scope.TraceError(err)
		log.Error().Err(err).Msg("failed to delete todo")

		return response.WithError(c, err)
	}

	user, _ := ctx.Value(constant.ContextKeyUserID).(string)
	scope.AddEvent(fmt.Sprintf("Todo deleted successfully by user %s", user))

	return response.WithMessage(c, fiber.StatusOK, "Todo deleted successfully")
}
