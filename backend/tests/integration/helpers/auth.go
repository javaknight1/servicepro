//go:build integration

package helpers

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"golang.org/x/crypto/bcrypt"
)

// =============================================================================
// Test User Types
// =============================================================================

// TestUser represents a user for testing
type TestUser struct {
	ID        string
	Email     string
	Password  string
	FirstName string
	LastName  string
	Role      string
	CompanyID string
	Token     string // JWT token after authentication
}

// TestRole represents a role for testing
type TestRole struct {
	ID          string
	Name        string
	Permissions []string
}

// =============================================================================
// Auth Helper
// =============================================================================

// AuthHelper provides authentication utilities for tests
type AuthHelper struct {
	db        *gorm.DB
	jwtSecret string
}

// NewAuthHelper creates a new auth helper
func NewAuthHelper(db *gorm.DB, jwtSecret string) *AuthHelper {
	return &AuthHelper{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

// =============================================================================
// User Management
// =============================================================================

// CreateTestUser creates a test user in the database
func (h *AuthHelper) CreateTestUser(t *testing.T, user *TestUser) *TestUser {
	t.Helper()

	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	if user.Email == "" {
		user.Email = "test-" + uuid.New().String()[:8] + "@test.com"
	}
	if user.Password == "" {
		user.Password = "TestPassword123!"
	}
	if user.FirstName == "" {
		user.FirstName = "Test"
	}
	if user.LastName == "" {
		user.LastName = "User"
	}
	if user.Role == "" {
		user.Role = "user"
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	require.NoError(t, err, "Failed to hash password")

	// Insert user into database
	err = h.db.Exec(`
		INSERT INTO users (id, email, password_hash, first_name, last_name, role, email_verified, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, true, NOW(), NOW())
	`, user.ID, user.Email, string(hashedPassword), user.FirstName, user.LastName, user.Role).Error
	require.NoError(t, err, "Failed to create test user")

	return user
}

// CreateAdminUser creates an admin test user
func (h *AuthHelper) CreateAdminUser(t *testing.T) *TestUser {
	t.Helper()
	return h.CreateTestUser(t, &TestUser{
		Role:      "admin",
		FirstName: "Admin",
		LastName:  "User",
	})
}

// CreateRegularUser creates a regular test user
func (h *AuthHelper) CreateRegularUser(t *testing.T) *TestUser {
	t.Helper()
	return h.CreateTestUser(t, &TestUser{
		Role:      "user",
		FirstName: "Regular",
		LastName:  "User",
	})
}

// CreateManagerUser creates a manager test user
func (h *AuthHelper) CreateManagerUser(t *testing.T) *TestUser {
	t.Helper()
	return h.CreateTestUser(t, &TestUser{
		Role:      "manager",
		FirstName: "Manager",
		LastName:  "User",
	})
}

// CreateTechnicianUser creates a technician test user
func (h *AuthHelper) CreateTechnicianUser(t *testing.T) *TestUser {
	t.Helper()
	return h.CreateTestUser(t, &TestUser{
		Role:      "technician",
		FirstName: "Tech",
		LastName:  "User",
	})
}

// =============================================================================
// JWT Token Generation
// =============================================================================

// GenerateToken generates a JWT token for the user
func (h *AuthHelper) GenerateToken(t *testing.T, user *TestUser) string {
	t.Helper()

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"role":  user.Role,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	require.NoError(t, err, "Failed to generate JWT token")

	user.Token = tokenString
	return tokenString
}

// GenerateExpiredToken generates an expired JWT token
func (h *AuthHelper) GenerateExpiredToken(t *testing.T, user *TestUser) string {
	t.Helper()

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"role":  user.Role,
		"exp":   time.Now().Add(-time.Hour).Unix(), // Expired 1 hour ago
		"iat":   time.Now().Add(-2 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	require.NoError(t, err, "Failed to generate expired JWT token")

	return tokenString
}

// GenerateInvalidToken generates an invalid JWT token (wrong secret)
func (h *AuthHelper) GenerateInvalidToken(t *testing.T, user *TestUser) string {
	t.Helper()

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"role":  user.Role,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("wrong-secret"))
	require.NoError(t, err, "Failed to generate invalid JWT token")

	return tokenString
}

// GenerateRefreshToken generates a refresh token
func (h *AuthHelper) GenerateRefreshToken(t *testing.T, user *TestUser) string {
	t.Helper()

	claims := jwt.MapClaims{
		"sub":  user.ID,
		"type": "refresh",
		"exp":  time.Now().Add(24 * 7 * time.Hour).Unix(), // 7 days
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	require.NoError(t, err, "Failed to generate refresh token")

	return tokenString
}

// =============================================================================
// Authenticated Test Client
// =============================================================================

// AuthenticatedClient creates an HTTP client with authentication
func (h *AuthHelper) AuthenticatedClient(t *testing.T, baseURL string, user *TestUser) *TestClient {
	t.Helper()

	client := NewTestClient(baseURL)
	token := h.GenerateToken(t, user)
	client.SetAuthToken(token)
	return client
}

// =============================================================================
// Permission Helpers
// =============================================================================

// CreateRole creates a test role
func (h *AuthHelper) CreateRole(t *testing.T, role *TestRole) *TestRole {
	t.Helper()

	if role.ID == "" {
		role.ID = uuid.New().String()
	}

	err := h.db.Exec(`
		INSERT INTO roles (id, name, created_at, updated_at)
		VALUES (?, ?, NOW(), NOW())
	`, role.ID, role.Name).Error
	require.NoError(t, err, "Failed to create test role")

	// Add permissions
	for _, perm := range role.Permissions {
		var permID string
		err := h.db.Raw("SELECT id FROM permissions WHERE name = ?", perm).Scan(&permID).Error
		if err == nil && permID != "" {
			h.db.Exec(`
				INSERT INTO role_permissions (role_id, permission_id)
				VALUES (?, ?)
				ON CONFLICT DO NOTHING
			`, role.ID, permID)
		}
	}

	return role
}

// AssignRole assigns a role to a user
func (h *AuthHelper) AssignRole(t *testing.T, userID, roleName string) {
	t.Helper()

	err := h.db.Exec("UPDATE users SET role = ? WHERE id = ?", roleName, userID).Error
	require.NoError(t, err, "Failed to assign role")
}

// =============================================================================
// Company/Tenant Helpers
// =============================================================================

// CreateCompany creates a test company/tenant
func (h *AuthHelper) CreateCompany(t *testing.T, name string) string {
	t.Helper()

	companyID := uuid.New().String()
	err := h.db.Exec(`
		INSERT INTO companies (id, name, created_at, updated_at)
		VALUES (?, ?, NOW(), NOW())
	`, companyID, name).Error
	require.NoError(t, err, "Failed to create test company")

	return companyID
}

// AssignUserToCompany assigns a user to a company
func (h *AuthHelper) AssignUserToCompany(t *testing.T, userID, companyID string) {
	t.Helper()

	err := h.db.Exec("UPDATE users SET company_id = ? WHERE id = ?", companyID, userID).Error
	require.NoError(t, err, "Failed to assign user to company")
}
