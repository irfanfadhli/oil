package dto

import (
	"mime/multipart"

	"oil/internal/domains/room/model"
	"oil/shared"
	gDto "oil/shared/dto"
	gModel "oil/shared/model"
	"oil/shared/timezone"

	"github.com/google/uuid"
)

type CreateRoomRequest struct {
	Name      string                `json:"name"     validate:"required,max=100"`
	Location  string                `json:"location" validate:"omitempty,max=100"`
	Capacity  int                   `json:"capacity" validate:"omitempty,min=0"`
	Image     *multipart.FileHeader `json:"image"    validate:"omitempty,mimetypes=image/png image/jpg image/jpeg,maxfilesize=1"`
	ImageFile multipart.File        `json:"-"`
	Active    *bool                 `json:"active"   validate:"omitempty"`
}

func (c *CreateRoomRequest) ToModel(user string, imageURL string) model.Room {
	active := true
	if c.Active != nil {
		active = *c.Active
	}

	return model.Room{
		ID:       uuid.NewString(),
		Name:     c.Name,
		Location: c.Location,
		Capacity: c.Capacity,
		Image:    imageURL,
		Active:   active,
		Metadata: gModel.Metadata{
			CreatedAt:  timezone.Now(),
			ModifiedAt: timezone.Now(),
			CreatedBy:  user,
			ModifiedBy: user,
		},
	}
}

type UpdateRoomRequest struct {
	Name      string                `db:"name"     json:"name"                                                                 validate:"omitempty,max=100"`
	Location  string                `db:"location" json:"location"                                                             validate:"omitempty,max=100"`
	Capacity  *int                  `db:"capacity" json:"capacity"                                                             validate:"omitempty,min=0"`
	Image     *multipart.FileHeader `json:"image"  validate:"omitempty,mimetypes=image/png image/jpg image/jpeg,maxfilesize=1"`
	ImageFile multipart.File        `json:"-"`
	Active    *bool                 `db:"active"   json:"active"                                                               validate:"omitempty"`
}

type RoomResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
	Capacity int    `json:"capacity"`
	Image    string `json:"image"`
	Active   bool   `json:"active"`
	gDto.Metadata
}

func (r *RoomResponse) FromModel(model model.Room) {
	r.ID = model.ID
	r.Name = model.Name
	r.Location = model.Location
	r.Capacity = model.Capacity
	r.Image = model.Image
	r.Active = model.Active
	r.Metadata.FromModel(model.Metadata)
}

type GetRoomsResponse struct {
	Rooms     []RoomResponse `json:"rooms"`
	TotalPage int            `json:"total_page"`
	TotalData int            `json:"total_data"`
}

func (r *GetRoomsResponse) FromModels(models []model.Room, totalData, limit int) {
	r.TotalData = totalData
	r.TotalPage = shared.CalculateTotalPage(totalData, limit)

	r.Rooms = make([]RoomResponse, len(models))
	for i, mod := range models {
		r.Rooms[i].FromModel(mod)
	}
}
