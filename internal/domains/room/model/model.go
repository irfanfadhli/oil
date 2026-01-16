package model

import "oil/shared/model"

const (
	TableName  = "rooms"
	EntityName = "room"

	FieldID       = "id"
	FieldName     = "name"
	FieldLocation = "location"
	FieldCapacity = "capacity"
	FieldImage    = "image"
	FieldActive   = "active"
)

type Room struct {
	ID       string `db:"id"`
	Name     string `db:"name"`
	Location string `db:"location"`
	Capacity int    `db:"capacity"`
	Image    string `db:"image"`
	Active   bool   `db:"active"`
	model.Metadata
}
