package shared

import (
	"github.com/rs/zerolog/log"
	"math"
	"oil/shared/constant"
	"oil/shared/dto"
	"oil/shared/timezone"
	"reflect"
	"strconv"
)

func ConvertStringToBool(value string) *bool {
	if value == "" {
		return nil
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		log.Error().Err(err).Msg("failed to convert string to bool")

		return nil
	}

	return &boolValue
}

func CalculateTotalPage(total, limit int) (res int) {
	if total == 0 || limit <= 0 {
		res = 1
	} else {
		res = int(math.Ceil(float64(total) / float64(limit)))
	}

	return res
}

// TransformFields converts the fields of a struct into a map of updated fields.
func TransformFields(data interface{}, username string) map[string]any {
	val := reflect.ValueOf(data)
	typ := reflect.TypeOf(data)

	updatedFields := make(map[string]any)

	for index := range val.NumField() {
		field := val.Field(index)
		if field.IsZero() {
			continue
		}

		fieldName := typ.Field(index).Tag.Get("db")
		if fieldName == "" {
			continue
		}

		updatedFields[fieldName] = field.Interface()
	}

	updatedFields[constant.FieldModifiedAt] = timezone.Now()
	updatedFields[constant.FieldModifiedBy] = username

	return updatedFields
}

func FilterByID(id, fieldID, table string) dto.FilterGroup {
	return dto.FilterGroup{
		Filters: []any{
			dto.Filter{
				Field:    fieldID,
				Value:    id,
				Operator: dto.FilterOperatorEq,
				Table:    table,
			},
		},
	}
}
