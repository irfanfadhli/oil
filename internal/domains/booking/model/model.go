package model

import (
	"oil/shared/model"
	"time"
)

const (
	TableName  = "room_bookings"
	EntityName = "booking"

	FieldID          = "id"
	FieldRoomID      = "room_id"
	FieldGuestName   = "guest_name"
	FieldGuestEmail  = "guest_email"
	FieldGuestPhone  = "guest_phone"
	FieldBookingDate = "booking_date"
	FieldStartTime   = "start_time"
	FieldEndTime     = "end_time"
	FieldPurpose     = "purpose"
	FieldStatus      = "status"
	FieldCreatedBy   = "created_by"
)

type Booking struct {
	ID          string    `db:"id"`
	RoomID      string    `db:"room_id"`
	GuestName   string    `db:"guest_name"`
	GuestEmail  string    `db:"guest_email"`
	GuestPhone  string    `db:"guest_phone"`
	BookingDate time.Time `db:"booking_date"`
	StartTime   time.Time `db:"start_time"`
	EndTime     time.Time `db:"end_time"`
	Purpose     string    `db:"purpose"`
	Status      string    `db:"status"`
	model.Metadata
}
