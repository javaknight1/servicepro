package services

import (
	"context"
	"errors"
	"log"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
	"github.com/javaknight1/servicepro/backend/pkg/clients/email"
)

var (
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrWeakPassword       = errors.New("password does not meet strength requirements")
)

// RegistrationService handles user registration business logic
type RegistrationService struct {
	userRepo            repository.UserRepositoryInterface
	emailClient         email.Client
	verificationService EmailVerificationServiceInterface
}

// NewRegistrationService creates a new registration service
func NewRegistrationService(
	userRepo repository.UserRepositoryInterface,
	emailClient email.Client,
	verificationService EmailVerificationServiceInterface,
) *RegistrationService {
	return &RegistrationService{
		userRepo:            userRepo,
		emailClient:         emailClient,
		verificationService: verificationService,
	}
}

// Register registers a new user
func (s *RegistrationService) Register(emailAddr, password string) (*models.RegisterResponse, error) {
	// Validate email format
	if err := auth.ValidateEmail(emailAddr); err != nil {
		return nil, err
	}

	// Validate password strength
	if err := auth.ValidatePassword(password); err != nil {
		return nil, ErrWeakPassword
	}

	// Check if email already exists
	if gormRepo, ok := s.userRepo.(*repository.UserRepositoryGorm); ok {
		exists, err := gormRepo.EmailExists(emailAddr)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrEmailAlreadyExists
		}
	} else {
		// Fallback for non-GORM repositories
		_, err := s.userRepo.GetByEmail(emailAddr)
		if err == nil {
			return nil, ErrEmailAlreadyExists
		}
		if !errors.Is(err, repository.ErrUserNotFound) {
			return nil, err
		}
	}

	// Hash password
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Create user
	user, err := s.userRepo.Create(emailAddr, passwordHash)
	if err != nil {
		return nil, err
	}

	// Send email verification (async, don't fail registration if email fails)
	go func() {
		if s.verificationService != nil {
			if err := s.verificationService.SendVerificationEmail(user.ID); err != nil {
				log.Printf("Failed to send verification email to %s: %v", user.Email, err)
			}
		} else {
			// Fallback to welcome email if verification service not configured
			if err := s.emailClient.SendWelcomeEmail(context.Background(), user.Email, user.Email); err != nil {
				log.Printf("Failed to send welcome email to %s: %v", user.Email, err)
			}
		}
	}()

	// Return response
	return &models.RegisterResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}
