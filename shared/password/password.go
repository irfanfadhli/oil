package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost is the default cost for bcrypt hashing
	DefaultCost = bcrypt.DefaultCost
)

var (
	ErrInvalidPassword = errors.New("invalid password")
)

// Hash generates a bcrypt hash of the password
func Hash(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(bytes), nil
}

// Verify checks if the provided password matches the hash
func Verify(password, hash string) error {
	if password == "" || hash == "" {
		return ErrInvalidPassword
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidPassword
		}
		return fmt.Errorf("failed to verify password: %w", err)
	}

	return nil
}
