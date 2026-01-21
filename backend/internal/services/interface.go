package services

import (
	"github.com/javaknight1/servicepro/backend/internal/models"
)

// AuthServiceInterface defines the interface for auth service
type AuthServiceInterface interface {
	Login(email, password string) (*models.LoginResponse, error)
	RefreshToken(refreshToken string) (*models.LoginResponse, error)
}
