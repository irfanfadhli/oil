package model

import "oil/shared/model"

const (
	TableName  = "galleries"
	EntityName = "gallery"

	FieldID          = "id"
	FieldTitle       = "title"
	FieldDescription = "description"
	FieldImages      = "images"
)

type Gallery struct {
	ID          string   `db:"id"`
	Title       string   `db:"title"`
	Description string   `db:"description"`
	Images      []string `db:"images"`
	model.Metadata
}
