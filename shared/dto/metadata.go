package dto

import (
	"oil/shared/constant"
	"oil/shared/model"
	"oil/shared/timezone"
)

type Metadata struct {
	CreatedAt  string `json:"created_at"`
	ModifiedAt string `json:"modified_at"`
	CreatedBy  string `json:"created_by"`
	ModifiedBy string `json:"modified_by"`
}

func (m *Metadata) FromModel(model model.Metadata) {
	m.CreatedAt = timezone.Format(model.CreatedAt, constant.DateFormat)
	m.ModifiedAt = timezone.Format(model.ModifiedAt, constant.DateFormat)
	m.CreatedBy = model.CreatedBy
	m.ModifiedBy = model.ModifiedBy
}
