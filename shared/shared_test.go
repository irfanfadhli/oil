package shared_test

import (
	"oil/shared"
	"oil/shared/constant"
	"oil/shared/dto"
	"reflect"
	"testing"
	"time"
)

func TestConvertStringToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *bool
	}{
		{
			name:     "empty string returns nil",
			input:    "",
			expected: nil,
		},
		{
			name:     "valid true string",
			input:    "true",
			expected: boolPtr(true),
		},
		{
			name:     "valid false string",
			input:    "false",
			expected: boolPtr(false),
		},
		{
			name:     "valid 1 string",
			input:    "1",
			expected: boolPtr(true),
		},
		{
			name:     "valid 0 string",
			input:    "0",
			expected: boolPtr(false),
		},
		{
			name:     "valid t string",
			input:    "t",
			expected: boolPtr(true),
		},
		{
			name:     "valid f string",
			input:    "f",
			expected: boolPtr(false),
		},
		{
			name:     "valid T string",
			input:    "T",
			expected: boolPtr(true),
		},
		{
			name:     "valid F string",
			input:    "F",
			expected: boolPtr(false),
		},
		{
			name:     "valid TRUE string",
			input:    "TRUE",
			expected: boolPtr(true),
		},
		{
			name:     "valid FALSE string",
			input:    "FALSE",
			expected: boolPtr(false),
		},
		{
			name:     "invalid string returns nil",
			input:    "invalid",
			expected: nil,
		},
		{
			name:     "random string returns nil",
			input:    "random",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shared.ConvertStringToBool(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", *result)
				}
			} else {
				if result == nil {
					t.Errorf("expected %v, got nil", *tt.expected)
				} else if *result != *tt.expected {
					t.Errorf("expected %v, got %v", *tt.expected, *result)
				}
			}
		})
	}
}

func TestCalculateTotalPage(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		limit    int
		expected int
	}{
		{
			name:     "zero total returns 1",
			total:    0,
			limit:    10,
			expected: 1,
		},
		{
			name:     "zero limit returns 1",
			total:    100,
			limit:    0,
			expected: 1,
		},
		{
			name:     "negative limit returns 1",
			total:    100,
			limit:    -5,
			expected: 1,
		},
		{
			name:     "exact division",
			total:    100,
			limit:    10,
			expected: 10,
		},
		{
			name:     "division with remainder",
			total:    101,
			limit:    10,
			expected: 11,
		},
		{
			name:     "single item",
			total:    1,
			limit:    10,
			expected: 1,
		},
		{
			name:     "limit equals total",
			total:    10,
			limit:    10,
			expected: 1,
		},
		{
			name:     "limit greater than total",
			total:    5,
			limit:    10,
			expected: 1,
		},
		{
			name:     "large numbers",
			total:    1000000,
			limit:    7,
			expected: 142858,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shared.CalculateTotalPage(tt.total, tt.limit)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestTransformFields(t *testing.T) {
	type TestStruct struct {
		ID         int    `db:"id"`
		Name       string `db:"name"`
		Email      string `db:"email"`
		EmptyField string `db:"empty_field"`
		NoDBTag    string
		IgnoredTag string `db:"-"`
		NoTagField string `db:""`
	}

	tests := []struct {
		name     string
		data     interface{}
		username string
		expected map[string]any
	}{
		{
			name: "struct with populated fields",
			data: TestStruct{
				ID:         1,
				Name:       "John Doe",
				Email:      "john@example.com",
				EmptyField: "",        // zero value, should be ignored
				NoDBTag:    "ignored", // no db tag, should be ignored
				IgnoredTag: "ignored", // db:"-", should be ignored
				NoTagField: "ignored", // db:"", should be ignored
			},
			username: "testuser",
			expected: map[string]any{
				"id":    1,
				"name":  "John Doe",
				"email": "john@example.com",
				"-":     "ignored", // This will be included because db:"-" is not empty
				// EmptyField should not be included (zero value)
				// NoDBTag should not be included (no db tag)
				// NoTagField should not be included (db:"")
			},
		},
		{
			name:     "struct with all zero values",
			data:     TestStruct{},
			username: "testuser",
			expected: map[string]any{
				// Only metadata fields should be present
			},
		},
		{
			name: "struct with partial fields",
			data: TestStruct{
				Name: "Jane Doe",
				// Other fields are zero values
			},
			username: "admin",
			expected: map[string]any{
				"name": "Jane Doe",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shared.TransformFields(tt.data, tt.username)

			// Check that modified_at and modified_by are always set
			if result[constant.FieldModifiedAt] == nil {
				t.Error("expected modified_at to be set")
			}
			if result[constant.FieldModifiedBy] != tt.username {
				t.Errorf("expected modified_by to be %s, got %v", tt.username, result[constant.FieldModifiedBy])
			}

			// Check that modified_at is a time.Time
			if _, ok := result[constant.FieldModifiedAt].(time.Time); !ok {
				t.Error("expected modified_at to be a time.Time")
			}

			// Check expected fields (excluding metadata fields)
			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("expected field %s to exist", key)
				} else if !reflect.DeepEqual(actualValue, expectedValue) {
					t.Errorf("expected field %s to be %v, got %v", key, expectedValue, actualValue)
				}
			}

			// Check that no unexpected fields exist (excluding metadata fields)
			for key := range result {
				if key == constant.FieldModifiedAt || key == constant.FieldModifiedBy {
					continue // Skip metadata fields
				}
				if _, expected := tt.expected[key]; !expected {
					t.Errorf("unexpected field %s in result", key)
				}
			}
		})
	}
}

func TestTransformFieldsWithPointers(t *testing.T) {
	type TestStructWithPointers struct {
		ID    *int    `db:"id"`
		Name  *string `db:"name"`
		Count *int    `db:"count"`
	}

	name := "John"
	count := 0 // This is not a zero value for *int (nil is)

	data := TestStructWithPointers{
		ID:    intPtr(1),
		Name:  &name,
		Count: &count, // Should be included even though value is 0
	}

	result := shared.TransformFields(data, "testuser")

	expectedFields := map[string]any{
		"id":    intPtr(1),
		"name":  &name,
		"count": &count,
	}

	for key, expectedValue := range expectedFields {
		if actualValue, exists := result[key]; !exists {
			t.Errorf("expected field %s to exist", key)
		} else if !reflect.DeepEqual(actualValue, expectedValue) {
			t.Errorf("expected field %s to be %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestFilterByID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		fieldID  string
		table    string
		expected dto.FilterGroup
	}{
		{
			name:    "basic filter by id",
			id:      "123",
			fieldID: "user_id",
			table:   "users",
			expected: dto.FilterGroup{
				Filters: []any{
					dto.Filter{
						Field:    "user_id",
						Value:    "123",
						Operator: dto.FilterOperatorEq,
						Table:    "users",
					},
				},
			},
		},
		{
			name:    "filter with empty table",
			id:      "456",
			fieldID: "id",
			table:   "",
			expected: dto.FilterGroup{
				Filters: []any{
					dto.Filter{
						Field:    "id",
						Value:    "456",
						Operator: dto.FilterOperatorEq,
						Table:    "",
					},
				},
			},
		},
		{
			name:    "filter with uuid",
			id:      "550e8400-e29b-41d4-a716-446655440000",
			fieldID: "uuid",
			table:   "products",
			expected: dto.FilterGroup{
				Filters: []any{
					dto.Filter{
						Field:    "uuid",
						Value:    "550e8400-e29b-41d4-a716-446655440000",
						Operator: dto.FilterOperatorEq,
						Table:    "products",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shared.FilterByID(tt.id, tt.fieldID, tt.table)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}

			// Additional checks for the filter structure
			if len(result.Filters) != 1 {
				t.Errorf("expected 1 filter, got %d", len(result.Filters))
			}

			filter, ok := result.Filters[0].(dto.Filter)
			if !ok {
				t.Error("expected filter to be of type dto.Filter")
			}

			if filter.Field != tt.fieldID {
				t.Errorf("expected field to be %s, got %s", tt.fieldID, filter.Field)
			}

			if filter.Value != tt.id {
				t.Errorf("expected value to be %s, got %v", tt.id, filter.Value)
			}

			if filter.Operator != dto.FilterOperatorEq {
				t.Errorf("expected operator to be %s, got %s", dto.FilterOperatorEq, filter.Operator)
			}

			if filter.Table != tt.table {
				t.Errorf("expected table to be %s, got %s", tt.table, filter.Table)
			}
		})
	}
}

// Helper functions for creating pointers
func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}
