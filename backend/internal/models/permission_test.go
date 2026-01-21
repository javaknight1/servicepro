package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPermission_Validate(t *testing.T) {
	tests := []struct {
		name       string
		permission Permission
		wantErr    error
	}{
		{
			name: "Valid permission",
			permission: Permission{
				Name:     "users.create",
				Resource: "users",
				Action:   "create",
			},
			wantErr: nil,
		},
		{
			name: "Empty name",
			permission: Permission{
				Name:     "",
				Resource: "users",
				Action:   "create",
			},
			wantErr: ErrPermissionNameRequired,
		},
		{
			name: "Name too long",
			permission: Permission{
				Name:     string(make([]byte, 101)),
				Resource: "users",
				Action:   "create",
			},
			wantErr: ErrPermissionNameTooLong,
		},
		{
			name: "Empty resource",
			permission: Permission{
				Name:     "users.create",
				Resource: "",
				Action:   "create",
			},
			wantErr: ErrResourceRequired,
		},
		{
			name: "Empty action",
			permission: Permission{
				Name:     "users.create",
				Resource: "users",
				Action:   "",
			},
			wantErr: ErrActionRequired,
		},
		{
			name: "Invalid action",
			permission: Permission{
				Name:     "users.invalid",
				Resource: "users",
				Action:   "invalid_action",
			},
			wantErr: ErrInvalidAction,
		},
		{
			name: "Valid action - create",
			permission: Permission{
				Name:     "users.create",
				Resource: "users",
				Action:   "create",
			},
			wantErr: nil,
		},
		{
			name: "Valid action - read",
			permission: Permission{
				Name:     "users.read",
				Resource: "users",
				Action:   "read",
			},
			wantErr: nil,
		},
		{
			name: "Valid action - update",
			permission: Permission{
				Name:     "users.update",
				Resource: "users",
				Action:   "update",
			},
			wantErr: nil,
		},
		{
			name: "Valid action - delete",
			permission: Permission{
				Name:     "users.delete",
				Resource: "users",
				Action:   "delete",
			},
			wantErr: nil,
		},
		{
			name: "Valid action - list",
			permission: Permission{
				Name:     "users.list",
				Resource: "users",
				Action:   "list",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.permission.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidAction(t *testing.T) {
	tests := []struct {
		name   string
		action string
		want   bool
	}{
		{"create", "create", true},
		{"read", "read", true},
		{"update", "update", true},
		{"delete", "delete", true},
		{"list", "list", true},
		{"manage", "manage", true},
		{"execute", "execute", true},
		{"approve", "approve", true},
		{"assign", "assign", true},
		{"grant", "grant", true},
		{"invalid", "invalid_action", false},
		{"empty", "", false},
		{"uppercase", "CREATE", true}, // Should be case-insensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidAction(tt.action))
		})
	}
}

func TestParsePermissionName(t *testing.T) {
	tests := []struct {
		name         string
		permName     string
		wantResource string
		wantAction   string
		wantErr      error
	}{
		{
			name:         "Valid format",
			permName:     "users.create",
			wantResource: "users",
			wantAction:   "create",
			wantErr:      nil,
		},
		{
			name:         "Valid format with multiple dots",
			permName:     "api.users.create", // Should still split on first dot
			wantResource: "api",
			wantAction:   "users.create",
			wantErr:      nil,
		},
		{
			name:         "Invalid format - no dot",
			permName:     "userscreate",
			wantResource: "",
			wantAction:   "",
			wantErr:      ErrInvalidPermissionFormat,
		},
		{
			name:         "Invalid format - empty",
			permName:     "",
			wantResource: "",
			wantAction:   "",
			wantErr:      ErrInvalidPermissionFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, action, err := ParsePermissionName(tt.permName)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResource, resource)
				assert.Equal(t, tt.wantAction, action)
			}
		})
	}
}

func TestPermission_Matches(t *testing.T) {
	perm := Permission{
		Name:     "users.create",
		Resource: "users",
		Action:   "create",
	}

	tests := []struct {
		name     string
		resource string
		action   string
		want     bool
	}{
		{"Exact match", "users", "create", true},
		{"Wrong resource", "orders", "create", false},
		{"Wrong action", "users", "read", false},
		{"Both wrong", "orders", "read", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, perm.Matches(tt.resource, tt.action))
		})
	}
}

func TestPermission_MatchesName(t *testing.T) {
	perm := Permission{
		Name:     "users.create",
		Resource: "users",
		Action:   "create",
	}

	tests := []struct {
		name     string
		permName string
		want     bool
	}{
		{"Match", "users.create", true},
		{"No match", "users.read", false},
		{"No match", "orders.create", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, perm.MatchesName(tt.permName))
		})
	}
}

func TestPermission_GetFullName(t *testing.T) {
	tests := []struct {
		name       string
		permission Permission
		want       string
	}{
		{
			name: "Name provided",
			permission: Permission{
				Name:     "users.create",
				Resource: "users",
				Action:   "create",
			},
			want: "users.create",
		},
		{
			name: "Name not provided - generate from resource.action",
			permission: Permission{
				Name:     "",
				Resource: "orders",
				Action:   "read",
			},
			want: "orders.read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.permission.GetFullName())
		})
	}
}

func TestPermission_IsWildcard(t *testing.T) {
	tests := []struct {
		name   string
		action string
		want   bool
	}{
		{"Wildcard", "*", true},
		{"Not wildcard - create", "create", false},
		{"Not wildcard - read", "read", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm := Permission{Action: tt.action}
			assert.Equal(t, tt.want, perm.IsWildcard())
		})
	}
}

func TestPermission_CoversAction(t *testing.T) {
	tests := []struct {
		name        string
		permAction  string
		checkAction string
		want        bool
	}{
		{"Exact match", "create", "create", true},
		{"No match", "create", "read", false},
		{"Wildcard covers all", "*", "create", true},
		{"Wildcard covers read", "*", "read", true},
		{"Wildcard covers delete", "*", "delete", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm := Permission{Action: tt.permAction}
			assert.Equal(t, tt.want, perm.CoversAction(tt.checkAction))
		})
	}
}

func TestPermission_CoversResource(t *testing.T) {
	tests := []struct {
		name          string
		permResource  string
		checkResource string
		want          bool
	}{
		{"Exact match", "users", "users", true},
		{"No match", "users", "orders", false},
		{"Wildcard covers all", "*", "users", true},
		{"Wildcard covers orders", "*", "orders", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm := Permission{Resource: tt.permResource}
			assert.Equal(t, tt.want, perm.CoversResource(tt.checkResource))
		})
	}
}

func TestPermission_Covers(t *testing.T) {
	tests := []struct {
		name       string
		permission Permission
		resource   string
		action     string
		want       bool
	}{
		{
			name:       "Exact match",
			permission: Permission{Resource: "users", Action: "create"},
			resource:   "users",
			action:     "create",
			want:       true,
		},
		{
			name:       "Wildcard resource",
			permission: Permission{Resource: "*", Action: "create"},
			resource:   "users",
			action:     "create",
			want:       true,
		},
		{
			name:       "Wildcard action",
			permission: Permission{Resource: "users", Action: "*"},
			resource:   "users",
			action:     "create",
			want:       true,
		},
		{
			name:       "Both wildcard",
			permission: Permission{Resource: "*", Action: "*"},
			resource:   "users",
			action:     "create",
			want:       true,
		},
		{
			name:       "Wrong resource",
			permission: Permission{Resource: "users", Action: "create"},
			resource:   "orders",
			action:     "create",
			want:       false,
		},
		{
			name:       "Wrong action",
			permission: Permission{Resource: "users", Action: "create"},
			resource:   "users",
			action:     "read",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.permission.Covers(tt.resource, tt.action))
		})
	}
}

func TestPermission_ToResponse(t *testing.T) {
	permID := uuid.New()
	now := time.Now()

	perm := Permission{
		ID:          permID,
		Name:        "users.create",
		Description: "Create users",
		Resource:    "users",
		Action:      "create",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	resp := perm.ToResponse()

	assert.Equal(t, permID, resp.ID)
	assert.Equal(t, "users.create", resp.Name)
	assert.Equal(t, "Create users", resp.Description)
	assert.Equal(t, "users", resp.Resource)
	assert.Equal(t, "create", resp.Action)
	assert.Equal(t, now, resp.CreatedAt)
	assert.Equal(t, now, resp.UpdatedAt)
}

func TestPermissionChecker_Has(t *testing.T) {
	perm1 := Permission{ID: uuid.New(), Name: "users.create"}
	perm2 := Permission{ID: uuid.New(), Name: "users.read"}

	checker := NewPermissionChecker([]Permission{perm1, perm2})

	tests := []struct {
		name           string
		permissionName string
		want           bool
	}{
		{"Has users.create", "users.create", true},
		{"Has users.read", "users.read", true},
		{"No users.delete", "users.delete", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, checker.Has(tt.permissionName))
		})
	}
}

func TestPermissionChecker_Can(t *testing.T) {
	perm1 := Permission{ID: uuid.New(), Name: "users.create", Resource: "users", Action: "create"}
	perm2 := Permission{ID: uuid.New(), Name: "users.read", Resource: "users", Action: "read"}

	checker := NewPermissionChecker([]Permission{perm1, perm2})

	tests := []struct {
		name     string
		resource string
		action   string
		want     bool
	}{
		{"Can users.create", "users", "create", true},
		{"Can users.read", "users", "read", true},
		{"Cannot users.delete", "users", "delete", false},
		{"Cannot orders.create", "orders", "create", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, checker.Can(tt.resource, tt.action))
		})
	}
}

func TestPermissionChecker_CanAny(t *testing.T) {
	perm1 := Permission{ID: uuid.New(), Name: "users.create"}
	perm2 := Permission{ID: uuid.New(), Name: "users.read"}

	checker := NewPermissionChecker([]Permission{perm1, perm2})

	tests := []struct {
		name        string
		permissions []string
		want        bool
	}{
		{"Has one permission", []string{"users.create", "users.delete"}, true},
		{"Has both permissions", []string{"users.create", "users.read"}, true},
		{"Has none", []string{"users.delete", "users.update"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, checker.CanAny(tt.permissions...))
		})
	}
}

func TestPermissionChecker_CanAll(t *testing.T) {
	perm1 := Permission{ID: uuid.New(), Name: "users.create"}
	perm2 := Permission{ID: uuid.New(), Name: "users.read"}

	checker := NewPermissionChecker([]Permission{perm1, perm2})

	tests := []struct {
		name        string
		permissions []string
		want        bool
	}{
		{"Has all", []string{"users.create", "users.read"}, true},
		{"Has one of two", []string{"users.create", "users.delete"}, false},
		{"Has none", []string{"users.delete", "users.update"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, checker.CanAll(tt.permissions...))
		})
	}
}

func TestPermissionChecker_GetPermissionsByResource(t *testing.T) {
	perm1 := Permission{ID: uuid.New(), Name: "users.create", Resource: "users", Action: "create"}
	perm2 := Permission{ID: uuid.New(), Name: "users.read", Resource: "users", Action: "read"}
	perm3 := Permission{ID: uuid.New(), Name: "orders.create", Resource: "orders", Action: "create"}

	checker := NewPermissionChecker([]Permission{perm1, perm2, perm3})

	userPerms := checker.GetPermissionsByResource("users")
	assert.Len(t, userPerms, 2, "Should have 2 user permissions")

	orderPerms := checker.GetPermissionsByResource("orders")
	assert.Len(t, orderPerms, 1, "Should have 1 order permission")

	nonePerms := checker.GetPermissionsByResource("products")
	assert.Len(t, nonePerms, 0, "Should have 0 product permissions")
}

func TestPermissionChecker_GetPermissionsByAction(t *testing.T) {
	perm1 := Permission{ID: uuid.New(), Name: "users.create", Resource: "users", Action: "create"}
	perm2 := Permission{ID: uuid.New(), Name: "orders.create", Resource: "orders", Action: "create"}
	perm3 := Permission{ID: uuid.New(), Name: "users.read", Resource: "users", Action: "read"}

	checker := NewPermissionChecker([]Permission{perm1, perm2, perm3})

	createPerms := checker.GetPermissionsByAction("create")
	assert.Len(t, createPerms, 2, "Should have 2 create permissions")

	readPerms := checker.GetPermissionsByAction("read")
	assert.Len(t, readPerms, 1, "Should have 1 read permission")

	deletePerms := checker.GetPermissionsByAction("delete")
	assert.Len(t, deletePerms, 0, "Should have 0 delete permissions")
}

func TestPermissionChecker_GroupByResource(t *testing.T) {
	perm1 := Permission{ID: uuid.New(), Name: "users.create", Resource: "users", Action: "create"}
	perm2 := Permission{ID: uuid.New(), Name: "users.read", Resource: "users", Action: "read"}
	perm3 := Permission{ID: uuid.New(), Name: "orders.create", Resource: "orders", Action: "create"}

	checker := NewPermissionChecker([]Permission{perm1, perm2, perm3})

	grouped := checker.GroupByResource()

	assert.Len(t, grouped, 2, "Should have 2 resource groups")

	// Check that we have groups for users and orders
	resourceMap := make(map[string][]Permission)
	for _, group := range grouped {
		resourceMap[group.Resource] = group.Permissions
	}

	assert.Len(t, resourceMap["users"], 2, "Users should have 2 permissions")
	assert.Len(t, resourceMap["orders"], 1, "Orders should have 1 permission")
}
