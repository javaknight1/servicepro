package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRole_Validate(t *testing.T) {
	tests := []struct {
		name    string
		role    Role
		wantErr error
	}{
		{
			name: "Valid role",
			role: Role{
				Name:           "test_role",
				Description:    "Test role",
				HierarchyLevel: 50,
			},
			wantErr: nil,
		},
		{
			name: "Empty name",
			role: Role{
				Name:           "",
				HierarchyLevel: 50,
			},
			wantErr: ErrRoleNameRequired,
		},
		{
			name: "Name too long",
			role: Role{
				Name:           string(make([]byte, 101)), // 101 characters
				HierarchyLevel: 50,
			},
			wantErr: ErrRoleNameTooLong,
		},
		{
			name: "Invalid hierarchy level - negative",
			role: Role{
				Name:           "test_role",
				HierarchyLevel: -1,
			},
			wantErr: ErrInvalidHierarchyLevel,
		},
		{
			name: "Invalid hierarchy level - too high",
			role: Role{
				Name:           "test_role",
				HierarchyLevel: 101,
			},
			wantErr: ErrInvalidHierarchyLevel,
		},
		{
			name: "Valid boundary - hierarchy level 0",
			role: Role{
				Name:           "test_role",
				HierarchyLevel: 0,
			},
			wantErr: nil,
		},
		{
			name: "Valid boundary - hierarchy level 100",
			role: Role{
				Name:           "test_role",
				HierarchyLevel: 100,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.role.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRole_IsSystemRole(t *testing.T) {
	tests := []struct {
		name           string
		hierarchyLevel int
		want           bool
	}{
		{"Super admin", 100, true},
		{"Admin", 80, true},
		{"Manager", 60, false},
		{"User", 40, false},
		{"Guest", 20, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := Role{HierarchyLevel: tt.hierarchyLevel}
			assert.Equal(t, tt.want, role.IsSystemRole())
		})
	}
}

func TestRole_IsSuperAdmin(t *testing.T) {
	tests := []struct {
		name           string
		roleName       string
		hierarchyLevel int
		want           bool
	}{
		{"Super admin by name", "super_admin", 100, true},
		{"Super admin by level", "custom_super", 100, true},
		{"Admin", "admin", 80, false},
		{"User", "user", 40, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := Role{
				Name:           tt.roleName,
				HierarchyLevel: tt.hierarchyLevel,
			}
			assert.Equal(t, tt.want, role.IsSuperAdmin())
		})
	}
}

func TestRole_IsAdmin(t *testing.T) {
	tests := []struct {
		name           string
		hierarchyLevel int
		want           bool
	}{
		{"Super admin", 100, true},
		{"Admin", 80, true},
		{"Manager", 60, false},
		{"User", 40, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := Role{HierarchyLevel: tt.hierarchyLevel}
			assert.Equal(t, tt.want, role.IsAdmin())
		})
	}
}

func TestRole_HasPermission(t *testing.T) {
	perm1 := Permission{ID: uuid.New(), Name: "users.create"}
	perm2 := Permission{ID: uuid.New(), Name: "users.read"}

	role := Role{
		Name:        "test_role",
		Permissions: []Permission{perm1, perm2},
	}

	tests := []struct {
		name           string
		permissionName string
		want           bool
	}{
		{"Has permission", "users.create", true},
		{"Has permission 2", "users.read", true},
		{"No permission", "users.delete", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, role.HasPermission(tt.permissionName))
		})
	}
}

func TestRole_HasPermissionForResource(t *testing.T) {
	perm1 := Permission{ID: uuid.New(), Name: "users.create", Resource: "users", Action: "create"}
	perm2 := Permission{ID: uuid.New(), Name: "users.read", Resource: "users", Action: "read"}

	role := Role{
		Name:        "test_role",
		Permissions: []Permission{perm1, perm2},
	}

	tests := []struct {
		name     string
		resource string
		action   string
		want     bool
	}{
		{"Has users.create", "users", "create", true},
		{"Has users.read", "users", "read", true},
		{"No users.delete", "users", "delete", false},
		{"No orders.create", "orders", "create", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, role.HasPermissionForResource(tt.resource, tt.action))
		})
	}
}

func TestRole_GetAllPermissions(t *testing.T) {
	// Create permission hierarchy
	perm1 := Permission{ID: uuid.New(), Name: "users.create"}
	perm2 := Permission{ID: uuid.New(), Name: "users.read"}
	perm3 := Permission{ID: uuid.New(), Name: "users.update"}

	parentRole := Role{
		ID:          uuid.New(),
		Name:        "parent_role",
		Permissions: []Permission{perm1, perm2},
	}

	childRole := Role{
		ID:           uuid.New(),
		Name:         "child_role",
		ParentRoleID: &parentRole.ID,
		ParentRole:   &parentRole,
		Permissions:  []Permission{perm3},
	}

	// Get all permissions (should include parent permissions)
	allPerms := childRole.GetAllPermissions()

	assert.Len(t, allPerms, 3, "Should have 3 permissions total")

	// Check that all permissions are present
	permNames := make(map[string]bool)
	for _, perm := range allPerms {
		permNames[perm.Name] = true
	}

	assert.True(t, permNames["users.create"], "Should have users.create from parent")
	assert.True(t, permNames["users.read"], "Should have users.read from parent")
	assert.True(t, permNames["users.update"], "Should have users.update from self")
}

func TestRole_HasPermissionRecursive(t *testing.T) {
	perm1 := Permission{ID: uuid.New(), Name: "users.create"}
	perm2 := Permission{ID: uuid.New(), Name: "users.read"}

	parentRole := Role{
		ID:          uuid.New(),
		Name:        "parent_role",
		Permissions: []Permission{perm1},
	}

	childRole := Role{
		ID:           uuid.New(),
		Name:         "child_role",
		ParentRoleID: &parentRole.ID,
		ParentRole:   &parentRole,
		Permissions:  []Permission{perm2},
	}

	tests := []struct {
		name           string
		permissionName string
		want           bool
	}{
		{"Has own permission", "users.read", true},
		{"Has parent permission", "users.create", true},
		{"No permission", "users.delete", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, childRole.HasPermissionRecursive(tt.permissionName))
		})
	}
}

func TestRole_IsHigherThan(t *testing.T) {
	admin := Role{HierarchyLevel: 80}
	manager := Role{HierarchyLevel: 60}
	user := Role{HierarchyLevel: 40}

	assert.True(t, admin.IsHigherThan(&manager), "Admin should be higher than manager")
	assert.True(t, admin.IsHigherThan(&user), "Admin should be higher than user")
	assert.True(t, manager.IsHigherThan(&user), "Manager should be higher than user")
	assert.False(t, user.IsHigherThan(&admin), "User should not be higher than admin")
}

func TestRole_IsLowerThan(t *testing.T) {
	admin := Role{HierarchyLevel: 80}
	user := Role{HierarchyLevel: 40}

	assert.True(t, user.IsLowerThan(&admin), "User should be lower than admin")
	assert.False(t, admin.IsLowerThan(&user), "Admin should not be lower than user")
}

func TestRole_CanManage(t *testing.T) {
	admin := Role{HierarchyLevel: 80}
	manager := Role{HierarchyLevel: 60}
	user := Role{HierarchyLevel: 40}

	assert.True(t, admin.CanManage(&manager), "Admin can manage manager")
	assert.True(t, admin.CanManage(&user), "Admin can manage user")
	assert.True(t, manager.CanManage(&user), "Manager can manage user")
	assert.False(t, user.CanManage(&admin), "User cannot manage admin")
	assert.False(t, manager.CanManage(&admin), "Manager cannot manage admin")
}

func TestRole_GetDescendants(t *testing.T) {
	// Create role hierarchy: parent -> child1 -> grandchild
	//                                -> child2
	grandchildID := uuid.New()
	grandchild := Role{
		ID:   grandchildID,
		Name: "grandchild",
	}

	child1ID := uuid.New()
	child1 := Role{
		ID:         child1ID,
		Name:       "child1",
		ChildRoles: []Role{grandchild},
	}

	child2ID := uuid.New()
	child2 := Role{
		ID:   child2ID,
		Name: "child2",
	}

	parent := Role{
		ID:         uuid.New(),
		Name:       "parent",
		ChildRoles: []Role{child1, child2},
	}

	// Update parent references
	child1.ParentRoleID = &parent.ID
	child2.ParentRoleID = &parent.ID
	grandchild.ParentRoleID = &child1.ID

	// Get descendants
	descendants := parent.GetDescendants()

	assert.Len(t, descendants, 3, "Should have 3 descendants")

	// Check that all descendants are present
	descendantNames := make(map[string]bool)
	for _, desc := range descendants {
		descendantNames[desc.Name] = true
	}

	assert.True(t, descendantNames["child1"], "Should include child1")
	assert.True(t, descendantNames["child2"], "Should include child2")
	assert.True(t, descendantNames["grandchild"], "Should include grandchild")
}

func TestRole_GetAncestors(t *testing.T) {
	// Create hierarchy: grandparent -> parent -> child
	grandparent := Role{
		ID:   uuid.New(),
		Name: "grandparent",
	}

	parent := Role{
		ID:           uuid.New(),
		Name:         "parent",
		ParentRoleID: &grandparent.ID,
		ParentRole:   &grandparent,
	}

	child := Role{
		ID:           uuid.New(),
		Name:         "child",
		ParentRoleID: &parent.ID,
		ParentRole:   &parent,
	}

	ancestors := child.GetAncestors()

	assert.Len(t, ancestors, 2, "Should have 2 ancestors")
	assert.Equal(t, "parent", ancestors[0].Name, "First ancestor should be parent")
	assert.Equal(t, "grandparent", ancestors[1].Name, "Second ancestor should be grandparent")
}

func TestRole_ValidateHierarchy_NoCircular(t *testing.T) {
	grandparent := Role{
		ID:   uuid.New(),
		Name: "grandparent",
	}

	parent := Role{
		ID:           uuid.New(),
		Name:         "parent",
		ParentRoleID: &grandparent.ID,
		ParentRole:   &grandparent,
	}

	child := Role{
		ID:           uuid.New(),
		Name:         "child",
		ParentRoleID: &parent.ID,
		ParentRole:   &parent,
	}

	err := child.ValidateHierarchy()
	assert.NoError(t, err, "Valid hierarchy should not error")
}

func TestRole_ValidateHierarchy_Circular(t *testing.T) {
	role1 := Role{
		ID:   uuid.New(),
		Name: "role1",
	}

	role2 := Role{
		ID:           uuid.New(),
		Name:         "role2",
		ParentRoleID: &role1.ID,
		ParentRole:   &role1,
	}

	// Create circular reference
	role1.ParentRoleID = &role2.ID
	role1.ParentRole = &role2

	err := role2.ValidateHierarchy()
	assert.ErrorIs(t, err, ErrCircularRoleHierarchy, "Circular hierarchy should error")
}

func TestRoleAssignment_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{"No expiration", nil, false},
		{"Expired", &past, true},
		{"Not expired", &future, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ra := RoleAssignment{
				ExpiresAt: tt.expiresAt,
			}
			assert.Equal(t, tt.want, ra.IsExpired())
		})
	}
}

func TestRoleAssignment_IsActive(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{"No expiration - active", nil, true},
		{"Expired - not active", &past, false},
		{"Not expired - active", &future, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ra := RoleAssignment{
				ExpiresAt: tt.expiresAt,
			}
			assert.Equal(t, tt.want, ra.IsActive())
		})
	}
}

func TestRole_ToResponse(t *testing.T) {
	roleID := uuid.New()
	parentID := uuid.New()
	perm1 := Permission{
		ID:       uuid.New(),
		Name:     "users.create",
		Resource: "users",
		Action:   "create",
	}

	role := Role{
		ID:             roleID,
		Name:           "test_role",
		Description:    "Test role",
		HierarchyLevel: 50,
		ParentRoleID:   &parentID,
		Permissions:    []Permission{perm1},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	resp := role.ToResponse()

	assert.Equal(t, roleID, resp.ID)
	assert.Equal(t, "test_role", resp.Name)
	assert.Equal(t, "Test role", resp.Description)
	assert.Equal(t, 50, resp.HierarchyLevel)
	assert.Equal(t, &parentID, resp.ParentRoleID)
	assert.Len(t, resp.Permissions, 1)
	assert.Equal(t, "users.create", resp.Permissions[0].Name)
}
