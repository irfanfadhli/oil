package dto

import (
	"oil/internal/domains/user/model"
	gDto "oil/shared/dto"
)

// UpdateUserRequest represents the request for updating a user (admin)
type UpdateUserRequest struct {
	Email        *string `json:"email" validate:"omitempty,email"`
	FullName     *string `json:"full_name" validate:"omitempty,min=2,max=100"`
	Level        *string `json:"level" validate:"omitempty,oneof=admin user"`
	Active       *bool   `json:"active"`
	ProfileImage *string `json:"profile_image"`
	IsVerified   *bool   `json:"is_verified"`
}

// UpdateProfileRequest represents the request for updating user profile (self)
type UpdateProfileRequest struct {
	Email        *string `json:"email" validate:"omitempty,email"`
	FullName     *string `json:"full_name" validate:"omitempty,min=2,max=100"`
	ProfileImage *string `json:"profile_image"`
}

// UserResponse represents the response for a single user
type UserResponse struct {
	ID           string  `json:"id"`
	Email        string  `json:"email"`
	FullName     *string `json:"full_name"`
	Level        string  `json:"level"`
	ProfileImage *string `json:"profile_image"`
	IsVerified   bool    `json:"is_verified"`
	LastLogin    *string `json:"last_login"`
	Active       bool    `json:"active"`
	gDto.Metadata
}

// FromModel converts a user model to UserResponse
func (r *UserResponse) FromModel(user model.User) {
	r.ID = user.ID
	r.Email = user.Email
	r.FullName = user.FullName
	r.Level = user.Level
	r.ProfileImage = user.ProfileImage
	r.IsVerified = user.IsVerified
	r.LastLogin = user.LastLogin
	r.Active = user.Active
	r.Metadata.FromModel(user.Metadata)
}

// GetUsersResponse represents the response for getting multiple users
type GetUsersResponse struct {
	Users []UserResponse `json:"users"`
	Total int64          `json:"total"`
	Count int            `json:"count"`
}

// FromModels converts user models to GetUsersResponse
func (r *GetUsersResponse) FromModels(users []model.User, total int64, limit int) {
	r.Users = make([]UserResponse, len(users))
	for i, user := range users {
		r.Users[i].FromModel(user)
	}
	r.Total = total
	r.Count = len(users)
}
