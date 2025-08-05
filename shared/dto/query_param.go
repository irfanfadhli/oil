package dto

import (
	"github.com/gofiber/fiber/v2"
	"oil/shared/constant"
	"strconv"
)

const (
	SortDirAsc  = "ASC"
	SortDirDesc = "DESC"
)

type QueryParams struct {
	Page    int    `json:"page"     validate:"omitempty"`
	Limit   int    `json:"limit"    validate:"omitempty"`
	SortBy  string `json:"sort_by"  validate:"omitempty"`
	SortDir string `json:"sort_dir" validate:"omitempty,oneof=ASC DESC"`
}

// FromRequest populates QueryParams from the HTTP request.
// It's recommended to call this method with `defaultRequest` set to true if data is large
// Example:
//
//	q := &dto.QueryParams{}
//	q.FromRequest(req, true)
//
// This will set default values for Page, Limit, SortBy, and SortDir if they are not provided in the request.
// If `defaultRequest` is false, it will only populate the fields that are present in the request.
func (q *QueryParams) FromRequest(ctx *fiber.Ctx, defaultRequest bool) {
	if page := ctx.Query(constant.RequestParamPage); page != "" {
		if pageInt, err := strconv.Atoi(page); err == nil && pageInt > 0 {
			q.Page = pageInt
		}
	}

	if limit := ctx.Query(constant.RequestParamLimit); limit != "" {
		if limitInt, err := strconv.Atoi(limit); err == nil && limitInt > 0 {
			q.Limit = limitInt
		}
	}

	if sortBy := ctx.Query(constant.RequestParamSortBy); sortBy != "" {
		q.SortBy = sortBy
	}

	if sortDir := ctx.Query(constant.RequestParamSortBy); sortDir == SortDirAsc || sortDir == SortDirDesc {
		q.SortDir = sortDir
	}

	if defaultRequest {
		if q.Page == 0 {
			q.Page = constant.DefaultValuePage
		}
		if q.Limit == 0 {
			q.Limit = constant.DefaultValueLimit
		}
		if q.SortBy == "" {
			q.SortBy = constant.DefaultValueSortBy
		}
		if q.SortDir == "" {
			q.SortDir = constant.DefaultValueSortDir
		}
	}
}
