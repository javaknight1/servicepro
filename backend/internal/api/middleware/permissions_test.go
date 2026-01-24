package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/services/permissions"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
)

// setupTestEnvironment creates all necessary test infrastructure
func setupTestEnvironment(t *testing.T) (*gorm.DB, *redis.Client, *miniredis.Miniredis, *auth.JWTManager, *PermissionMiddleware) {
	// Setup database (separate for each test)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// Manually create tables for testing (SQLite doesn't support UUID defaults)
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT DEFAULT 'user',
			profile_picture_url TEXT,
			email_verified INTEGER DEFAULT 0,
			verification_sent_at DATETIME,
			failed_login_count INTEGER DEFAULT 0,
			last_failed_login_at DATETIME,
			locked_until DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			parent_role_id TEXT,
			hierarchy_level INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS permissions (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			resource TEXT NOT NULL,
			action TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS role_permissions (
			role_id TEXT NOT NULL,
			permission_id TEXT NOT NULL,
			granted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			granted_by TEXT,
			PRIMARY KEY (role_id, permission_id)
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_roles (
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			assigned_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			assigned_by TEXT,
			expires_at DATETIME,
			PRIMARY KEY (user_id, role_id)
		)
	`).Error
	require.NoError(t, err)

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tenant_users (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			is_active INTEGER DEFAULT 1,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error
	require.NoError(t, err)

	// Setup Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Setup JWT manager
	jwtConfig := &config.JWTConfig{
		Secret:            "test-secret-key-for-testing",
		AccessExpiration:  time.Hour,
		RefreshExpiration: 24 * time.Hour,
	}
	jwtManager := auth.NewJWTManager(jwtConfig)

	// Setup permission checker
	permRepo := repository.NewPermissionRepository(db)
	permChecker := permissions.NewPermissionChecker(permRepo, redisClient)

	// Setup middleware
	middleware := NewPermissionMiddleware(permChecker, jwtManager)

	return db, redisClient, mr, jwtManager, middleware
}

// createTestUser creates a test user in the database
func createTestUser(t *testing.T, db *gorm.DB, email string) models.User {
	user := models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: "hash",
	}
	err := db.Create(&user).Error
	require.NoError(t, err)
	return user
}

// createTestPermission creates a test permission
func createTestPermission(t *testing.T, db *gorm.DB, name, resource, action string) models.Permission {
	perm := models.Permission{
		ID:       uuid.New(),
		Name:     name,
		Resource: resource,
		Action:   action,
	}
	err := db.Create(&perm).Error
	require.NoError(t, err)
	return perm
}

// createTestRole creates a test role
func createTestRole(t *testing.T, db *gorm.DB, name string, level int) models.Role {
	role := models.Role{
		ID:             uuid.New(),
		Name:           name,
		HierarchyLevel: level,
	}
	err := db.Create(&role).Error
	require.NoError(t, err)
	return role
}

// assignPermissionToRole assigns a permission to a role
func assignPermissionToRole(t *testing.T, db *gorm.DB, roleID, permID uuid.UUID) {
	err := db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", roleID, permID).Error
	require.NoError(t, err)
}

// assignRoleToUser assigns a role to a user
func assignRoleToUser(t *testing.T, db *gorm.DB, userID, roleID uuid.UUID) {
	err := db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, roleID).Error
	require.NoError(t, err)
}

func TestRequireAuth(t *testing.T) {
	db, _, mr, jwtManager, middleware := setupTestEnvironment(t)
	defer mr.Close()

	gin.SetMode(gin.TestMode)

	user := createTestUser(t, db, "test@example.com")

	t.Run("Valid token", func(t *testing.T) {
		// Generate valid token
		token, err := jwtManager.GenerateAccessToken(user.ID.String(), user.Email)
		require.NoError(t, err)

		// Create test request
		router := gin.New()
		router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Missing authorization header", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", response.Error)
	})

	t.Run("Invalid token format", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Invalid token", func(t *testing.T) {
		router := gin.New()
		router.GET("/test", middleware.RequireAuth(), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestRequirePermission(t *testing.T) {
	db, _, mr, jwtManager, middleware := setupTestEnvironment(t)
	defer mr.Close()

	gin.SetMode(gin.TestMode)

	// Create test data
	user := createTestUser(t, db, "test@example.com")
	perm := createTestPermission(t, db, "users.read", "users", "read")
	role := createTestRole(t, db, "test_role", 50)
	assignPermissionToRole(t, db, role.ID, perm.ID)
	assignRoleToUser(t, db, user.ID, role.ID)

	token, err := jwtManager.GenerateAccessToken(user.ID.String(), user.Email)
	require.NoError(t, err)

	t.Run("User has required permission", func(t *testing.T) {
		router := gin.New()
		router.GET("/users",
			middleware.RequireAuth(),
			middleware.RequirePermission("users.read"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("User lacks required permission", func(t *testing.T) {
		router := gin.New()
		router.POST("/users",
			middleware.RequireAuth(),
			middleware.RequirePermission("users.create"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

		req := httptest.NewRequest(http.MethodPost, "/users", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response models.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "forbidden", response.Error)
	})

	t.Run("User not authenticated", func(t *testing.T) {
		router := gin.New()
		router.GET("/users",
			middleware.RequirePermission("users.read"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestRequireResourceAction(t *testing.T) {
	db, _, mr, jwtManager, middleware := setupTestEnvironment(t)
	defer mr.Close()

	gin.SetMode(gin.TestMode)

	// Create test data
	user := createTestUser(t, db, "test@example.com")
	perm := createTestPermission(t, db, "users.update", "users", "update")
	role := createTestRole(t, db, "test_role", 50)
	assignPermissionToRole(t, db, role.ID, perm.ID)
	assignRoleToUser(t, db, user.ID, role.ID)

	token, err := jwtManager.GenerateAccessToken(user.ID.String(), user.Email)
	require.NoError(t, err)

	t.Run("User can perform action", func(t *testing.T) {
		router := gin.New()
		router.PUT("/users/:id",
			middleware.RequireAuth(),
			middleware.RequireResourceAction("users", "update"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

		req := httptest.NewRequest(http.MethodPut, "/users/123", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("User cannot perform action", func(t *testing.T) {
		router := gin.New()
		router.DELETE("/users/:id",
			middleware.RequireAuth(),
			middleware.RequireResourceAction("users", "delete"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

		req := httptest.NewRequest(http.MethodDelete, "/users/123", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestRequireAnyPermission(t *testing.T) {
	db, _, mr, jwtManager, middleware := setupTestEnvironment(t)
	defer mr.Close()

	gin.SetMode(gin.TestMode)

	// Create test data
	user := createTestUser(t, db, "test@example.com")
	perm := createTestPermission(t, db, "users.read", "users", "read")
	role := createTestRole(t, db, "test_role", 50)
	assignPermissionToRole(t, db, role.ID, perm.ID)
	assignRoleToUser(t, db, user.ID, role.ID)

	token, err := jwtManager.GenerateAccessToken(user.ID.String(), user.Email)
	require.NoError(t, err)

	t.Run("User has one of required permissions", func(t *testing.T) {
		router := gin.New()
		router.GET("/dashboard",
			middleware.RequireAuth(),
			middleware.RequireAnyPermission("users.read", "admin.access"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

		req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("User has none of required permissions", func(t *testing.T) {
		router := gin.New()
		router.GET("/admin",
			middleware.RequireAuth(),
			middleware.RequireAnyPermission("admin.access", "super.admin"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestRequireAllPermissions(t *testing.T) {
	db, _, mr, jwtManager, middleware := setupTestEnvironment(t)
	defer mr.Close()

	gin.SetMode(gin.TestMode)

	// Create test data
	user := createTestUser(t, db, "test@example.com")
	perm1 := createTestPermission(t, db, "users.read", "users", "read")
	perm2 := createTestPermission(t, db, "users.create", "users", "create")
	role := createTestRole(t, db, "test_role", 50)
	assignPermissionToRole(t, db, role.ID, perm1.ID)
	assignPermissionToRole(t, db, role.ID, perm2.ID)
	assignRoleToUser(t, db, user.ID, role.ID)

	token, err := jwtManager.GenerateAccessToken(user.ID.String(), user.Email)
	require.NoError(t, err)

	t.Run("User has all required permissions", func(t *testing.T) {
		router := gin.New()
		router.GET("/users/manage",
			middleware.RequireAuth(),
			middleware.RequireAllPermissions("users.read", "users.create"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

		req := httptest.NewRequest(http.MethodGet, "/users/manage", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("User missing one required permission", func(t *testing.T) {
		router := gin.New()
		router.GET("/users/full-access",
			middleware.RequireAuth(),
			middleware.RequireAllPermissions("users.read", "users.delete"),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

		req := httptest.NewRequest(http.MethodGet, "/users/full-access", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestRequireAdmin(t *testing.T) {
	db, _, mr, jwtManager, middleware := setupTestEnvironment(t)
	defer mr.Close()

	gin.SetMode(gin.TestMode)

	// Create admin user
	adminUser := createTestUser(t, db, "admin@example.com")
	adminRole := createTestRole(t, db, "admin", 80)
	assignRoleToUser(t, db, adminUser.ID, adminRole.ID)
	adminToken, err := jwtManager.GenerateAccessToken(adminUser.ID.String(), adminUser.Email)
	require.NoError(t, err)

	// Create regular user
	regularUser := createTestUser(t, db, "user@example.com")
	userRole := createTestRole(t, db, "user", 40)
	assignRoleToUser(t, db, regularUser.ID, userRole.ID)
	userToken, err := jwtManager.GenerateAccessToken(regularUser.ID.String(), regularUser.Email)
	require.NoError(t, err)

	t.Run("Admin user can access", func(t *testing.T) {
		router := gin.New()
		router.GET("/admin/dashboard",
			middleware.RequireAuth(),
			middleware.RequireAdmin(),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

		req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Regular user cannot access", func(t *testing.T) {
		router := gin.New()
		router.GET("/admin/dashboard",
			middleware.RequireAuth(),
			middleware.RequireAdmin(),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

		req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", userToken))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestRequireSuperAdmin(t *testing.T) {
	db, _, mr, jwtManager, middleware := setupTestEnvironment(t)
	defer mr.Close()

	gin.SetMode(gin.TestMode)

	// Create super admin user
	superAdminUser := createTestUser(t, db, "superadmin@example.com")
	superAdminRole := createTestRole(t, db, "super_admin", 100)
	assignRoleToUser(t, db, superAdminUser.ID, superAdminRole.ID)
	superAdminToken, err := jwtManager.GenerateAccessToken(superAdminUser.ID.String(), superAdminUser.Email)
	require.NoError(t, err)

	// Create admin user (not super admin)
	adminUser := createTestUser(t, db, "admin@example.com")
	adminRole := createTestRole(t, db, "admin", 80)
	assignRoleToUser(t, db, adminUser.ID, adminRole.ID)
	adminToken, err := jwtManager.GenerateAccessToken(adminUser.ID.String(), adminUser.Email)
	require.NoError(t, err)

	t.Run("Super admin can access", func(t *testing.T) {
		router := gin.New()
		router.DELETE("/admin/system",
			middleware.RequireAuth(),
			middleware.RequireSuperAdmin(),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

		req := httptest.NewRequest(http.MethodDelete, "/admin/system", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", superAdminToken))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Admin cannot access super admin route", func(t *testing.T) {
		router := gin.New()
		router.DELETE("/admin/system",
			middleware.RequireAuth(),
			middleware.RequireSuperAdmin(),
			func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

		req := httptest.NewRequest(http.MethodDelete, "/admin/system", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestGetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Valid user ID in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		expectedID := uuid.New()
		c.Set("user_id", expectedID)

		userID, err := GetUserID(c)
		assert.NoError(t, err)
		assert.Equal(t, expectedID, userID)
	})

	t.Run("User ID not in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		userID, err := GetUserID(c)
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, userID)
	})
}

func TestGetUserEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Valid user email in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		expectedEmail := "test@example.com"
		c.Set("user_email", expectedEmail)

		email, err := GetUserEmail(c)
		assert.NoError(t, err)
		assert.Equal(t, expectedEmail, email)
	})

	t.Run("User email not in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		email, err := GetUserEmail(c)
		assert.Error(t, err)
		assert.Empty(t, email)
	})
}
