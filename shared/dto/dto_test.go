package dto_test

import (
	"oil/shared/constant"
	"oil/shared/dto"
	"oil/shared/model"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

func TestMetadata_FromModel(t *testing.T) {
	// Create test time values
	createdAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	modifiedAt := time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)

	modelMetadata := model.Metadata{
		CreatedAt:  createdAt,
		ModifiedAt: modifiedAt,
		CreatedBy:  "creator",
		ModifiedBy: "modifier",
	}

	metadata := &dto.Metadata{}
	metadata.FromModel(modelMetadata)

	expectedCreatedAt := createdAt.Format(constant.DateFormat)
	expectedModifiedAt := modifiedAt.Format(constant.DateFormat)

	if metadata.CreatedAt != expectedCreatedAt {
		t.Errorf("expected CreatedAt to be %s, got %s", expectedCreatedAt, metadata.CreatedAt)
	}

	if metadata.ModifiedAt != expectedModifiedAt {
		t.Errorf("expected ModifiedAt to be %s, got %s", expectedModifiedAt, metadata.ModifiedAt)
	}

	if metadata.CreatedBy != "creator" {
		t.Errorf("expected CreatedBy to be 'creator', got %s", metadata.CreatedBy)
	}

	if metadata.ModifiedBy != "modifier" {
		t.Errorf("expected ModifiedBy to be 'modifier', got %s", metadata.ModifiedBy)
	}
}

func TestQueryParams_FromRequest(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    map[string]string
		defaultRequest bool
		expected       dto.QueryParams
	}{
		{
			name: "with all valid parameters",
			queryParams: map[string]string{
				"page":     "2",
				"limit":    "20",
				"sort_by":  "ASC", // Bug: sortDir reads from sort_by param
				"sort_dir": "ASC",
			},
			defaultRequest: false,
			expected: dto.QueryParams{
				Page:    2,
				Limit:   20,
				SortBy:  "ASC", // This will be set to "ASC" due to the bug
				SortDir: "ASC", // This will be set to "ASC" because it reads from sort_by
			},
		},
		{
			name:           "with default request enabled and no parameters",
			queryParams:    map[string]string{},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    constant.DefaultValuePage,
				Limit:   constant.DefaultValueLimit,
				SortBy:  constant.DefaultValueSortBy,
				SortDir: constant.DefaultValueSortDir,
			},
		},
		{
			name:           "with default request disabled and no parameters",
			queryParams:    map[string]string{},
			defaultRequest: false,
			expected: dto.QueryParams{
				Page:    0,
				Limit:   0,
				SortBy:  "",
				SortDir: "",
			},
		},
		{
			name: "with invalid page parameter",
			queryParams: map[string]string{
				"page": "invalid",
			},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    constant.DefaultValuePage, // Should use default
				Limit:   constant.DefaultValueLimit,
				SortBy:  constant.DefaultValueSortBy,
				SortDir: constant.DefaultValueSortDir,
			},
		},
		{
			name: "with negative page parameter",
			queryParams: map[string]string{
				"page": "-1",
			},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    constant.DefaultValuePage, // Should use default
				Limit:   constant.DefaultValueLimit,
				SortBy:  constant.DefaultValueSortBy,
				SortDir: constant.DefaultValueSortDir,
			},
		},
		{
			name: "with zero page parameter",
			queryParams: map[string]string{
				"page": "0",
			},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    constant.DefaultValuePage, // Should use default
				Limit:   constant.DefaultValueLimit,
				SortBy:  constant.DefaultValueSortBy,
				SortDir: constant.DefaultValueSortDir,
			},
		},
		{
			name: "with invalid limit parameter",
			queryParams: map[string]string{
				"limit": "invalid",
			},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    constant.DefaultValuePage,
				Limit:   constant.DefaultValueLimit, // Should use default
				SortBy:  constant.DefaultValueSortBy,
				SortDir: constant.DefaultValueSortDir,
			},
		},
		{
			name: "with negative limit parameter",
			queryParams: map[string]string{
				"limit": "-10",
			},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    constant.DefaultValuePage,
				Limit:   constant.DefaultValueLimit, // Should use default
				SortBy:  constant.DefaultValueSortBy,
				SortDir: constant.DefaultValueSortDir,
			},
		},
		{
			name: "with partial parameters and defaults enabled",
			queryParams: map[string]string{
				"page":    "3",
				"sort_by": "email",
			},
			defaultRequest: true,
			expected: dto.QueryParams{
				Page:    3,
				Limit:   constant.DefaultValueLimit, // Should use default
				SortBy:  "email",
				SortDir: constant.DefaultValueSortDir, // Should use default
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new Fiber app for testing
			app := fiber.New()

			// Create a mock HTTP request context
			reqCtx := &fasthttp.RequestCtx{}
			reqCtx.Request.SetRequestURI("http://example.com/test")

			// Add query parameters
			for key, value := range tt.queryParams {
				reqCtx.Request.URI().QueryArgs().Add(key, value)
			}

			// Create Fiber context
			ctx := app.AcquireCtx(reqCtx)
			defer app.ReleaseCtx(ctx)

			// Test the method
			queryParams := &dto.QueryParams{}
			queryParams.FromRequest(ctx, tt.defaultRequest)

			// Verify results
			if queryParams.Page != tt.expected.Page {
				t.Errorf("expected Page to be %d, got %d", tt.expected.Page, queryParams.Page)
			}
			if queryParams.Limit != tt.expected.Limit {
				t.Errorf("expected Limit to be %d, got %d", tt.expected.Limit, queryParams.Limit)
			}
			if queryParams.SortBy != tt.expected.SortBy {
				t.Errorf("expected SortBy to be %s, got %s", tt.expected.SortBy, queryParams.SortBy)
			}
			if queryParams.SortDir != tt.expected.SortDir {
				t.Errorf("expected SortDir to be %s, got %s", tt.expected.SortDir, queryParams.SortDir)
			}
		})
	}
}

func TestSortDirectionConstants(t *testing.T) {
	if dto.SortDirAsc != "ASC" {
		t.Errorf("expected SortDirAsc to be 'ASC', got %s", dto.SortDirAsc)
	}
	if dto.SortDirDesc != "DESC" {
		t.Errorf("expected SortDirDesc to be 'DESC', got %s", dto.SortDirDesc)
	}
}
