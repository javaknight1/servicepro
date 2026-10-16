package permissions

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	// Use separate database for each test to avoid conflicts
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
			first_name TEXT,
			last_name TEXT,
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

	return db
}

// setupTestRedis creates an in-memory Redis server for testing
func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr
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

// createTestUser creates a test user
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

func TestPermissionChecker_CheckPermission(t *testing.T) {
	db := setupTestDB(t)
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	repo := repository.NewPermissionRepository(db)
	checker := NewPermissionChecker(repo, redisClient)

	// Create test data
	user := createTestUser(t, db, "test@example.com")
	perm1 := createTestPermission(t, db, "users.create", "users", "create")
	perm2 := createTestPermission(t, db, "users.read", "users", "read")
	role := createTestRole(t, db, "test_role", 50)

	// Assign permissions to role
	err := db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", role.ID, perm1.ID).Error
	require.NoError(t, err)
	err = db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", role.ID, perm2.ID).Error
	require.NoError(t, err)

	// Assign role to user
	err = db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, role.ID).Error
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("User has permission", func(t *testing.T) {
		hasPermission, err := checker.CheckPermission(ctx, user.ID, uuid.Nil, "users.create")
		assert.NoError(t, err)
		assert.True(t, hasPermission)
	})

	t.Run("User does not have permission", func(t *testing.T) {
		hasPermission, err := checker.CheckPermission(ctx, user.ID, uuid.Nil, "users.delete")
		assert.NoError(t, err)
		assert.False(t, hasPermission)
	})

	t.Run("Cache is used on second call", func(t *testing.T) {
		// First call - populates cache
		hasPermission, err := checker.CheckPermission(ctx, user.ID, uuid.Nil, "users.create")
		assert.NoError(t, err)
		assert.True(t, hasPermission)

		// Verify cache was populated
		cacheKey := checker.getCacheKey(user.ID, uuid.Nil)
		cached, err := redisClient.Get(ctx, cacheKey).Result()
		assert.NoError(t, err)
		assert.NotEmpty(t, cached)

		// Second call - should use cache
		hasPermission, err = checker.CheckPermission(ctx, user.ID, uuid.Nil, "users.create")
		assert.NoError(t, err)
		assert.True(t, hasPermission)
	})

	t.Run("Non-existent user", func(t *testing.T) {
		nonExistentID := uuid.New()
		hasPermission, err := checker.CheckPermission(ctx, nonExistentID, uuid.Nil, "users.create")
		assert.NoError(t, err)
		assert.False(t, hasPermission)
	})
}

func TestPermissionChecker_CheckResourceAction(t *testing.T) {
	db := setupTestDB(t)
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	repo := repository.NewPermissionRepository(db)
	checker := NewPermissionChecker(repo, redisClient)

	// Create test data
	user := createTestUser(t, db, "test@example.com")
	perm := createTestPermission(t, db, "users.update", "users", "update")
	role := createTestRole(t, db, "test_role", 50)

	// Assign permission to role
	err := db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", role.ID, perm.ID).Error
	require.NoError(t, err)

	// Assign role to user
	err = db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, role.ID).Error
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("User can perform action on resource", func(t *testing.T) {
		canPerform, err := checker.CheckResourceAction(ctx, user.ID, uuid.Nil, "users", "update")
		assert.NoError(t, err)
		assert.True(t, canPerform)
	})

	t.Run("User cannot perform action on resource", func(t *testing.T) {
		canPerform, err := checker.CheckResourceAction(ctx, user.ID, uuid.Nil, "users", "delete")
		assert.NoError(t, err)
		assert.False(t, canPerform)
	})
}

func TestPermissionChecker_CheckAnyPermission(t *testing.T) {
	db := setupTestDB(t)
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	repo := repository.NewPermissionRepository(db)
	checker := NewPermissionChecker(repo, redisClient)

	// Create test data
	user := createTestUser(t, db, "test@example.com")
	perm1 := createTestPermission(t, db, "users.read", "users", "read")
	role := createTestRole(t, db, "test_role", 50)

	// Assign permission to role
	err := db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", role.ID, perm1.ID).Error
	require.NoError(t, err)

	// Assign role to user
	err = db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, role.ID).Error
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("User has one of the permissions", func(t *testing.T) {
		hasAny, err := checker.CheckAnyPermission(ctx, user.ID, uuid.Nil, "users.read", "users.delete")
		assert.NoError(t, err)
		assert.True(t, hasAny)
	})

	t.Run("User has none of the permissions", func(t *testing.T) {
		hasAny, err := checker.CheckAnyPermission(ctx, user.ID, uuid.Nil, "users.delete", "users.create")
		assert.NoError(t, err)
		assert.False(t, hasAny)
	})
}

func TestPermissionChecker_CheckAllPermissions(t *testing.T) {
	db := setupTestDB(t)
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	repo := repository.NewPermissionRepository(db)
	checker := NewPermissionChecker(repo, redisClient)

	// Create test data
	user := createTestUser(t, db, "test@example.com")
	perm1 := createTestPermission(t, db, "users.read", "users", "read")
	perm2 := createTestPermission(t, db, "users.create", "users", "create")
	role := createTestRole(t, db, "test_role", 50)

	// Assign permissions to role
	err := db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", role.ID, perm1.ID).Error
	require.NoError(t, err)
	err = db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", role.ID, perm2.ID).Error
	require.NoError(t, err)

	// Assign role to user
	err = db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, role.ID).Error
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("User has all permissions", func(t *testing.T) {
		hasAll, err := checker.CheckAllPermissions(ctx, user.ID, uuid.Nil, "users.read", "users.create")
		assert.NoError(t, err)
		assert.True(t, hasAll)
	})

	t.Run("User missing one permission", func(t *testing.T) {
		hasAll, err := checker.CheckAllPermissions(ctx, user.ID, uuid.Nil, "users.read", "users.delete")
		assert.NoError(t, err)
		assert.False(t, hasAll)
	})
}

func TestPermissionChecker_IsAdmin(t *testing.T) {
	db := setupTestDB(t)
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	repo := repository.NewPermissionRepository(db)
	checker := NewPermissionChecker(repo, redisClient)

	ctx := context.Background()

	t.Run("User is admin (hierarchy >= 80)", func(t *testing.T) {
		user := createTestUser(t, db, "admin@example.com")
		adminRole := createTestRole(t, db, "admin", 80)
		err := db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, adminRole.ID).Error
		require.NoError(t, err)

		isAdmin, err := checker.IsAdmin(ctx, user.ID, uuid.Nil)
		assert.NoError(t, err)
		assert.True(t, isAdmin)
	})

	t.Run("User is not admin (hierarchy < 80)", func(t *testing.T) {
		user := createTestUser(t, db, "user@example.com")
		userRole := createTestRole(t, db, "user", 40)
		err := db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, userRole.ID).Error
		require.NoError(t, err)

		isAdmin, err := checker.IsAdmin(ctx, user.ID, uuid.Nil)
		assert.NoError(t, err)
		assert.False(t, isAdmin)
	})
}

func TestPermissionChecker_IsSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	repo := repository.NewPermissionRepository(db)
	checker := NewPermissionChecker(repo, redisClient)

	ctx := context.Background()

	t.Run("User is super admin (hierarchy = 100)", func(t *testing.T) {
		user := createTestUser(t, db, "superadmin@example.com")
		superAdminRole := createTestRole(t, db, "super_admin", 100)
		err := db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, superAdminRole.ID).Error
		require.NoError(t, err)

		isSuperAdmin, err := checker.IsSuperAdmin(ctx, user.ID)
		assert.NoError(t, err)
		assert.True(t, isSuperAdmin)
	})

	t.Run("User is not super admin (hierarchy < 100)", func(t *testing.T) {
		user := createTestUser(t, db, "admin@example.com")
		adminRole := createTestRole(t, db, "admin", 80)
		err := db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, adminRole.ID).Error
		require.NoError(t, err)

		isSuperAdmin, err := checker.IsSuperAdmin(ctx, user.ID)
		assert.NoError(t, err)
		assert.False(t, isSuperAdmin)
	})
}

func TestPermissionChecker_InvalidateCache(t *testing.T) {
	db := setupTestDB(t)
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	repo := repository.NewPermissionRepository(db)
	checker := NewPermissionChecker(repo, redisClient)

	// Create test data
	user := createTestUser(t, db, "test@example.com")
	perm := createTestPermission(t, db, "users.read", "users", "read")
	role := createTestRole(t, db, "test_role", 50)

	// Assign permission to role
	err := db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", role.ID, perm.ID).Error
	require.NoError(t, err)

	// Assign role to user
	err = db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, role.ID).Error
	require.NoError(t, err)

	ctx := context.Background()

	// Populate cache
	hasPermission, err := checker.CheckPermission(ctx, user.ID, uuid.Nil, "users.read")
	assert.NoError(t, err)
	assert.True(t, hasPermission)

	// Verify cache exists
	cacheKey := checker.getCacheKey(user.ID, uuid.Nil)
	cached, err := redisClient.Get(ctx, cacheKey).Result()
	assert.NoError(t, err)
	assert.NotEmpty(t, cached)

	// Invalidate cache
	err = checker.InvalidateUserPermissions(ctx, user.ID)
	assert.NoError(t, err)

	// Verify cache is gone
	_, err = redisClient.Get(ctx, cacheKey).Result()
	assert.Error(t, err) // Should be redis.Nil error
}

func TestPermissionChecker_RefreshPermissions(t *testing.T) {
	db := setupTestDB(t)
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	repo := repository.NewPermissionRepository(db)
	checker := NewPermissionChecker(repo, redisClient)

	// Create test data
	user := createTestUser(t, db, "test@example.com")
	perm := createTestPermission(t, db, "users.read", "users", "read")
	role := createTestRole(t, db, "test_role", 50)

	// Assign permission to role
	err := db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", role.ID, perm.ID).Error
	require.NoError(t, err)

	// Assign role to user
	err = db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, role.ID).Error
	require.NoError(t, err)

	ctx := context.Background()

	// Refresh permissions (should populate cache)
	err = checker.RefreshUserPermissions(ctx, user.ID, uuid.Nil)
	assert.NoError(t, err)

	// Verify cache exists
	cacheKey := checker.getCacheKey(user.ID, uuid.Nil)
	cached, err := redisClient.Get(ctx, cacheKey).Result()
	assert.NoError(t, err)
	assert.NotEmpty(t, cached)

	// Verify cached data is correct
	var permissions []models.Permission
	err = json.Unmarshal([]byte(cached), &permissions)
	assert.NoError(t, err)
	assert.Len(t, permissions, 1)
	assert.Equal(t, "users.read", permissions[0].Name)
}

func TestPermissionChecker_CacheTTL(t *testing.T) {
	db := setupTestDB(t)
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	repo := repository.NewPermissionRepository(db)
	checker := NewPermissionChecker(repo, redisClient)

	// Create test data
	user := createTestUser(t, db, "test@example.com")
	perm := createTestPermission(t, db, "users.read", "users", "read")
	role := createTestRole(t, db, "test_role", 50)

	// Assign permission to role
	err := db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", role.ID, perm.ID).Error
	require.NoError(t, err)

	// Assign role to user
	err = db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, role.ID).Error
	require.NoError(t, err)

	ctx := context.Background()

	// Populate cache
	_, err = checker.CheckPermission(ctx, user.ID, uuid.Nil, "users.read")
	assert.NoError(t, err)

	// Check TTL
	cacheKey := checker.getCacheKey(user.ID, uuid.Nil)
	ttl, err := redisClient.TTL(ctx, cacheKey).Result()
	assert.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, permissionsCacheTTL)
}

func TestPermissionChecker_ExpiredRoleAssignment(t *testing.T) {
	db := setupTestDB(t)
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	repo := repository.NewPermissionRepository(db)
	checker := NewPermissionChecker(repo, redisClient)

	// Create test data
	user := createTestUser(t, db, "test@example.com")
	perm := createTestPermission(t, db, "users.read", "users", "read")
	role := createTestRole(t, db, "test_role", 50)

	// Assign permission to role
	err := db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", role.ID, perm.ID).Error
	require.NoError(t, err)

	// Assign role to user with expiration in the past
	expiredTime := time.Now().Add(-1 * time.Hour)
	err = db.Exec("INSERT INTO user_roles (user_id, role_id, expires_at) VALUES (?, ?, ?)", user.ID, role.ID, expiredTime).Error
	require.NoError(t, err)

	ctx := context.Background()

	// User should not have permission because role assignment is expired
	hasPermission, err := checker.CheckPermission(ctx, user.ID, uuid.Nil, "users.read")
	assert.NoError(t, err)
	assert.False(t, hasPermission)
}

// TestPermissionChecker_TenantScoped verifies F-2: a role held in tenant B
// never grants permissions in tenant A, and the cache is keyed per tenant.
func TestPermissionChecker_TenantScoped(t *testing.T) {
	db := setupTestDB(t)
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	checker := NewPermissionChecker(repository.NewPermissionRepository(db), redisClient)
	ctx := context.Background()

	user := createTestUser(t, db, "member@example.com")
	perm := createTestPermission(t, db, "customers.delete", "customers", "delete")
	admin := createTestRole(t, db, "admin", 80)
	guest := createTestRole(t, db, "guest", 20)
	require.NoError(t, db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", admin.ID, perm.ID).Error)

	tenantA, tenantB := uuid.New(), uuid.New()
	require.NoError(t, db.Exec("INSERT INTO tenant_users (id, tenant_id, user_id, role_id, is_active) VALUES (?, ?, ?, ?, 1)", uuid.New(), tenantA, user.ID, guest.ID).Error)
	require.NoError(t, db.Exec("INSERT INTO tenant_users (id, tenant_id, user_id, role_id, is_active) VALUES (?, ?, ?, ?, 1)", uuid.New(), tenantB, user.ID, admin.ID).Error)

	ok, err := checker.CheckPermission(ctx, user.ID, tenantB, "customers.delete")
	require.NoError(t, err)
	assert.True(t, ok, "admin in tenant B")

	ok, err = checker.CheckPermission(ctx, user.ID, tenantA, "customers.delete")
	require.NoError(t, err)
	assert.False(t, ok, "admin role in tenant B must not apply in tenant A")

	ok, err = checker.CheckPermission(ctx, user.ID, uuid.Nil, "customers.delete")
	require.NoError(t, err)
	assert.False(t, ok, "tenant memberships must not apply without a tenant")

	isAdmin, err := checker.IsAdmin(ctx, user.ID, tenantA)
	require.NoError(t, err)
	assert.False(t, isAdmin)
	isAdmin, err = checker.IsAdmin(ctx, user.ID, tenantB)
	require.NoError(t, err)
	assert.True(t, isAdmin)

	isSuper, err := checker.IsSuperAdmin(ctx, user.ID)
	require.NoError(t, err)
	assert.False(t, isSuper)

	// Separate cache entries per tenant, and invalidation clears all of them.
	assert.True(t, mr.Exists(checker.getCacheKey(user.ID, tenantA)))
	assert.True(t, mr.Exists(checker.getCacheKey(user.ID, tenantB)))
	require.NoError(t, checker.InvalidateUserPermissions(ctx, user.ID))
	assert.False(t, mr.Exists(checker.getCacheKey(user.ID, tenantA)))
	assert.False(t, mr.Exists(checker.getCacheKey(user.ID, tenantB)))
}

// T230: the effective permission list is per tenant and matches CheckPermission.
func TestPermissionChecker_EffectivePermissionNames(t *testing.T) {
	db := setupTestDB(t)
	redisClient, mr := setupTestRedis(t)
	defer mr.Close()

	checker := NewPermissionChecker(repository.NewPermissionRepository(db), redisClient)
	ctx := context.Background()

	user := createTestUser(t, db, "member@example.com")
	read := createTestPermission(t, db, "customers.read", "customers", "read")
	del := createTestPermission(t, db, "customers.delete", "customers", "delete")
	platform := createTestPermission(t, db, "settings.read", "settings", "read")
	admin := createTestRole(t, db, "admin", 80)
	guest := createTestRole(t, db, "guest", 20)
	platformRole := createTestRole(t, db, "platform_reader", 10)
	for _, rp := range [][2]uuid.UUID{{admin.ID, del.ID}, {admin.ID, read.ID}, {guest.ID, read.ID}, {platformRole.ID, platform.ID}} {
		require.NoError(t, db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)", rp[0], rp[1]).Error)
	}
	require.NoError(t, db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", user.ID, platformRole.ID).Error)

	tenantA, tenantB := uuid.New(), uuid.New()
	require.NoError(t, db.Exec("INSERT INTO tenant_users (id, tenant_id, user_id, role_id, is_active) VALUES (?, ?, ?, ?, 1)", uuid.New(), tenantA, user.ID, guest.ID).Error)
	require.NoError(t, db.Exec("INSERT INTO tenant_users (id, tenant_id, user_id, role_id, is_active) VALUES (?, ?, ?, ?, 1)", uuid.New(), tenantB, user.ID, admin.ID).Error)

	namesA, err := checker.EffectivePermissionNames(ctx, user.ID, tenantA)
	require.NoError(t, err)
	assert.Equal(t, []string{"customers.read", "settings.read"}, namesA, "guest in A plus platform role; admin in B does not leak")

	namesB, err := checker.EffectivePermissionNames(ctx, user.ID, tenantB)
	require.NoError(t, err)
	assert.Equal(t, []string{"customers.delete", "customers.read", "settings.read"}, namesB, "sorted and de-duplicated")

	for _, name := range namesB {
		ok, err := checker.CheckPermission(ctx, user.ID, tenantB, name)
		require.NoError(t, err)
		assert.True(t, ok, name)
	}

	namesNone, err := checker.EffectivePermissionNames(ctx, user.ID, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, []string{"settings.read"}, namesNone, "a tenant the user is not in adds nothing")
}
