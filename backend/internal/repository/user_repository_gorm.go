package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// UserRepositoryGorm implements UserRepositoryInterface using GORM
type UserRepositoryGorm struct {
	db *gorm.DB
}

// NewUserRepositoryGorm creates a new GORM-based user repository
func NewUserRepositoryGorm(db *gorm.DB) *UserRepositoryGorm {
	return &UserRepositoryGorm{db: db}
}

// GetByEmail retrieves a user by email
func (r *UserRepositoryGorm) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepositoryGorm) GetByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// Create creates a new user
func (r *UserRepositoryGorm) Create(email, passwordHash string) (*models.User, error) {
	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		Role:         models.RoleUser,
	}

	if err := r.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

// IncrementFailedLoginCount increments the failed login count for a user
func (r *UserRepositoryGorm) IncrementFailedLoginCount(userID int) error {
	// This method signature uses int but we need UUID
	// For backward compatibility with existing tests, we'll need to handle this
	// For now, we'll implement a UUID version
	return errors.New("use IncrementFailedLoginCountByUUID instead")
}

// IncrementFailedLoginCountByUUID increments the failed login count for a user by UUID
func (r *UserRepositoryGorm) IncrementFailedLoginCountByUUID(userID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"failed_login_count":   gorm.Expr("failed_login_count + 1"),
			"last_failed_login_at": now,
			"updated_at":           now,
		}).Error
}

// LockAccount locks a user account until the specified time
func (r *UserRepositoryGorm) LockAccount(userID int, until time.Time) error {
	// This method signature uses int but we need UUID
	return errors.New("use LockAccountByUUID instead")
}

// LockAccountByUUID locks a user account until the specified time
func (r *UserRepositoryGorm) LockAccountByUUID(userID uuid.UUID, until time.Time) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"locked_until": until,
			"updated_at":   time.Now(),
		}).Error
}

// ResetFailedLoginCount resets the failed login count for a user
func (r *UserRepositoryGorm) ResetFailedLoginCount(userID int) error {
	// This method signature uses int but we need UUID
	return errors.New("use ResetFailedLoginCountByUUID instead")
}

// ResetFailedLoginCountByUUID resets the failed login count for a user by UUID
func (r *UserRepositoryGorm) ResetFailedLoginCountByUUID(userID uuid.UUID) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"failed_login_count":   0,
			"last_failed_login_at": nil,
			"locked_until":         nil,
			"updated_at":           time.Now(),
		}).Error
}

// EmailExists checks if an email already exists
func (r *UserRepositoryGorm) EmailExists(email string) (bool, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdatePassword updates the user's password hash
func (r *UserRepositoryGorm) UpdatePassword(userID uuid.UUID, passwordHash string) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"password_hash": passwordHash,
			"updated_at":    time.Now(),
		}).Error
}

// MarkEmailVerified marks the user's email as verified
func (r *UserRepositoryGorm) MarkEmailVerified(userID uuid.UUID) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"email_verified": true,
			"updated_at":     time.Now(),
		}).Error
}

// UpdateVerificationSentAt updates the timestamp when verification email was sent
func (r *UserRepositoryGorm) UpdateVerificationSentAt(userID uuid.UUID, sentAt *time.Time) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"verification_sent_at": sentAt,
			"updated_at":           time.Now(),
		}).Error
}

// GetUnverifiedUsersForReminder retrieves unverified users who need reminders
func (r *UserRepositoryGorm) GetUnverifiedUsersForReminder(threshold time.Duration) ([]*models.User, error) {
	var users []*models.User
	cutoffTime := time.Now().Add(-threshold)

	err := r.db.Where("email_verified = ? AND verification_sent_at IS NOT NULL AND verification_sent_at < ?",
		false, cutoffTime).
		Find(&users).Error

	if err != nil {
		return nil, err
	}

	return users, nil
}

// UpdateProfilePicture updates the user's profile picture URL
func (r *UserRepositoryGorm) UpdateProfilePicture(userID uuid.UUID, url *string) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"profile_picture_url": url,
			"updated_at":          time.Now(),
		}).Error
}

// UpdateProfile updates the user's first and last name
func (r *UserRepositoryGorm) UpdateProfile(userID uuid.UUID, firstName, lastName *string) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"first_name": firstName,
			"last_name":  lastName,
			"updated_at": time.Now(),
		}).Error
}
