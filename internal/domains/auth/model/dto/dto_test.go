package dto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"oil/infras/jwt"
	"oil/internal/domains/auth/model/dto"
	"oil/shared/timezone"
)

func TestLoginResponse_FromTokenPair(t *testing.T) {
	tokenPair := &jwt.TokenPair{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	}

	var response dto.LoginResponse
	response.FromTokenPair(tokenPair)

	assert.Equal(t, tokenPair.AccessToken, response.AccessToken)
	assert.Equal(t, tokenPair.RefreshToken, response.RefreshToken)
}

func TestRefreshTokenResponse_FromTokenPair(t *testing.T) {
	tokenPair := &jwt.TokenPair{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
	}

	var response dto.RefreshTokenResponse
	response.FromTokenPair(tokenPair)

	assert.Equal(t, tokenPair.AccessToken, response.AccessToken)
	assert.Equal(t, tokenPair.RefreshToken, response.RefreshToken)
}

func TestUpdateLastLoginRequest(t *testing.T) {
	now := timezone.Now()

	req := dto.UpdateLastLoginRequest{
		LastLogin: now,
	}

	assert.Equal(t, now, req.LastLogin)
}

func TestUpdatePasswordRequest(t *testing.T) {
	hashedPassword := "hashed-new-password"

	req := dto.UpdatePasswordRequest{
		Password: hashedPassword,
	}

	assert.Equal(t, hashedPassword, req.Password)
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
