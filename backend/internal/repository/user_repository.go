package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

// UserRepository handles user data access
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, failed_login_count,
		       last_failed_login_at, locked_until, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FailedLoginCount,
		&user.LastFailedLoginAt,
		&user.LockedUntil,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// IncrementFailedLoginCount increments the failed login count for a user
func (r *UserRepository) IncrementFailedLoginCount(userID int) error {
	query := `
		UPDATE users
		SET failed_login_count = failed_login_count + 1,
		    last_failed_login_at = $1,
		    updated_at = $1
		WHERE id = $2
	`

	now := time.Now()
	_, err := r.db.Exec(query, now, userID)
	return err
}

// LockAccount locks a user account until the specified time
func (r *UserRepository) LockAccount(userID int, until time.Time) error {
	query := `
		UPDATE users
		SET locked_until = $1,
		    updated_at = $2
		WHERE id = $3
	`

	now := time.Now()
	_, err := r.db.Exec(query, until, now, userID)
	return err
}

// ResetFailedLoginCount resets the failed login count for a user
func (r *UserRepository) ResetFailedLoginCount(userID int) error {
	query := `
		UPDATE users
		SET failed_login_count = 0,
		    last_failed_login_at = NULL,
		    locked_until = NULL,
		    updated_at = $1
		WHERE id = $2
	`

	now := time.Now()
	_, err := r.db.Exec(query, now, userID)
	return err
}

// Create creates a new user (useful for testing and user registration)
func (r *UserRepository) Create(email, passwordHash string) (*models.User, error) {
	query := `
		INSERT INTO users (email, password_hash, failed_login_count, created_at, updated_at)
		VALUES ($1, $2, 0, $3, $3)
		RETURNING id, email, password_hash, failed_login_count,
		          last_failed_login_at, locked_until, created_at, updated_at
	`

	now := time.Now()
	user := &models.User{}

	err := r.db.QueryRow(query, email, passwordHash, now).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FailedLoginCount,
		&user.LastFailedLoginAt,
		&user.LockedUntil,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}
