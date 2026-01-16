package dto

import (
	"github.com/google/uuid"
	"oil/internal/domains/booking/model"
	"oil/shared"
	gDto "oil/shared/dto"
	gModel "oil/shared/model"
	"oil/shared/timezone"
	"time"
)

type CreateBookingRequest struct {
	RoomID      string `json:"room_id"      validate:"required"`
	GuestName   string `json:"guest_name"   validate:"required,max=100"`
	GuestEmail  string `json:"guest_email"  validate:"omitempty,email,max=100"`
	GuestPhone  string `json:"guest_phone"  validate:"omitempty,max=20"`
	BookingDate string `json:"booking_date" validate:"required"`
	StartTime   string `json:"start_time"   validate:"required"`
	EndTime     string `json:"end_time"     validate:"required"`
	Purpose     string `json:"purpose"      validate:"omitempty"`
	Status      string `json:"status"       validate:"omitempty,oneof=pending confirmed cancelled"`
}

func (c *CreateBookingRequest) ToModel(user string) (model.Booking, error) {
	bookingDate, err := time.Parse("2006-01-02", c.BookingDate)
	if err != nil {
		return model.Booking{}, err
	}

	startTime, err := time.Parse("15:04", c.StartTime)
	if err != nil {
		return model.Booking{}, err
	}

	endTime, err := time.Parse("15:04", c.EndTime)
	if err != nil {
		return model.Booking{}, err
	}

	status := "pending"
	if c.Status != "" {
		status = c.Status
	}

	return model.Booking{
		ID:          uuid.NewString(),
		RoomID:      c.RoomID,
		GuestName:   c.GuestName,
		GuestEmail:  c.GuestEmail,
		GuestPhone:  c.GuestPhone,
		BookingDate: bookingDate,
		StartTime:   startTime,
		EndTime:     endTime,
		Purpose:     c.Purpose,
		Status:      status,
		Metadata: gModel.Metadata{
			CreatedAt:  timezone.Now(),
			ModifiedAt: timezone.Now(),
			CreatedBy:  user,
			ModifiedBy: user,
		},
	}, nil
}

type UpdateBookingRequest struct {
	GuestName   string `db:"guest_name"     json:"guest_name"    validate:"omitempty,max=100"`
	GuestEmail  string `db:"guest_email"    json:"guest_email"   validate:"omitempty,email,max=100"`
	GuestPhone  string `db:"guest_phone"    json:"guest_phone"   validate:"omitempty,max=20"`
	BookingDate string `json:"booking_date" validate:"omitempty"`
	StartTime   string `json:"start_time"   validate:"omitempty"`
	EndTime     string `json:"end_time"     validate:"omitempty"`
	Purpose     string `db:"purpose"        json:"purpose"       validate:"omitempty"`
	Status      string `db:"status"         json:"status"        validate:"omitempty,oneof=pending confirmed cancelled"`
}

type BookingResponse struct {
	ID          string `json:"id"`
	RoomID      string `json:"room_id"`
	GuestName   string `json:"guest_name"`
	GuestEmail  string `json:"guest_email"`
	GuestPhone  string `json:"guest_phone"`
	BookingDate string `json:"booking_date"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Purpose     string `json:"purpose"`
	Status      string `json:"status"`
	gDto.Metadata
}

func (r *BookingResponse) FromModel(model model.Booking) {
	r.ID = model.ID
	r.RoomID = model.RoomID
	r.GuestName = model.GuestName
	r.GuestEmail = model.GuestEmail
	r.GuestPhone = model.GuestPhone
	r.BookingDate = model.BookingDate.Format("2006-01-02")
	r.StartTime = model.StartTime.Format("15:04")
	r.EndTime = model.EndTime.Format("15:04")
	r.Purpose = model.Purpose
	r.Status = model.Status
	r.Metadata.FromModel(model.Metadata)
}

type GetBookingsResponse struct {
	Bookings  []BookingResponse `json:"bookings"`
	TotalPage int               `json:"total_page"`
	TotalData int               `json:"total_data"`
}

func (r *GetBookingsResponse) FromModels(models []model.Booking, totalData, limit int) {
	r.TotalData = totalData
	r.TotalPage = shared.CalculateTotalPage(totalData, limit)

	r.Bookings = make([]BookingResponse, len(models))
	for i, mod := range models {
		r.Bookings[i].FromModel(mod)
	}
}
