package services

import "github.com/javaknight1/servicepro/backend/internal/models"

// RegistrationServiceInterface defines the interface for registration service
type RegistrationServiceInterface interface {
	Register(email, password string) (*models.RegisterResponse, error)
}
