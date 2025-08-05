package model

import "time"

type Metadata struct {
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	CreatedBy  string    `db:"created_by"`
	ModifiedBy string    `db:"modified_by"`
}
