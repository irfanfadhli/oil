package jwt

import (
	"errors"
	"fmt"
	"oil/config"
	"oil/shared/timezone"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
	ErrInvalidClaim = errors.New("invalid token claim")
)

// TokenType represents the type of JWT token
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims represents the JWT claims structure
type Claims struct {
	UserID   string    `json:"user_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role,omitempty"`
	TokenID  string    `json:"token_id"`
	Type     TokenType `json:"type"`
	IssuedAt time.Time `json:"iat"`
	jwt.RegisteredClaims
}

// TokenPair represents access and refresh token pair
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// JWT handles JWT operations
type JWT interface {
	GenerateTokenPair(userID, email, role string) (*TokenPair, error)
	ValidateToken(tokenString string, tokenType TokenType) (*Claims, error)
	RefreshTokens(refreshToken string) (*TokenPair, error)
}

// Service handles JWT operations
type Service struct {
	config *config.Config
}

// New creates a new JWT service
func New(cfg *config.Config) JWT {
	return &Service{
		config: cfg,
	}
}

// GenerateTokenPair generates both access and refresh tokens
func (s *Service) GenerateTokenPair(userID, email, role string) (*TokenPair, error) {
	now := timezone.Now()

	// Generate access token
	accessToken, err := s.generateToken(userID, email, role, AccessToken, now, s.config.JWT.AccessExpireMin)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.generateToken(userID, email, role, RefreshToken, now, s.config.JWT.RefreshExpireMin)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.config.JWT.AccessExpireMin * 60),
	}, nil
}

// generateToken creates a JWT token with the specified parameters
func (s *Service) generateToken(userID, email, role string, tokenType TokenType, issuedAt time.Time, expireMin int) (string, error) {
	expiresAt := issuedAt.Add(time.Duration(expireMin) * time.Minute)
	tokenID := uuid.New().String()

	claims := Claims{
		UserID:   userID,
		Email:    email,
		Role:     role,
		TokenID:  tokenID,
		Type:     tokenType,
		IssuedAt: issuedAt,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			NotBefore: jwt.NewNumericDate(issuedAt),
			Issuer:    s.config.App.Name,
			Subject:   userID,
			ID:        tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	var secret string
	switch tokenType {
	case AccessToken:
		secret = s.config.JWT.AccessSecret
	case RefreshToken:
		secret = s.config.JWT.RefreshSecret
	default:
		return "", fmt.Errorf("unknown token type: %s", tokenType)
	}

	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates and parses a JWT token
func (s *Service) ValidateToken(tokenString string, tokenType TokenType) (*Claims, error) {
	var secret string
	switch tokenType {
	case AccessToken:
		secret = s.config.JWT.AccessSecret
	case RefreshToken:
		secret = s.config.JWT.RefreshSecret
	default:
		return nil, fmt.Errorf("unknown token type: %s", tokenType)
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Verify token type
	if claims.Type != tokenType {
		return nil, ErrInvalidClaim
	}

	return claims, nil
}

// RefreshTokens generates new token pair using refresh token
func (s *Service) RefreshTokens(refreshToken string) (*TokenPair, error) {
	claims, err := s.ValidateToken(refreshToken, RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Generate new token pair
	return s.GenerateTokenPair(claims.UserID, claims.Email, claims.Role)
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	const prefix = "Bearer "
	if len(authHeader) < len(prefix) || authHeader[:len(prefix)] != prefix {
		return "", errors.New("authorization header must start with 'Bearer '")
	}

	return authHeader[len(prefix):], nil
}
