package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func TestValidateAssignableRole(t *testing.T) {
	admin := MemberActor{UserID: uuid.New(), RoleLevel: 80}
	manager := MemberActor{UserID: uuid.New(), RoleLevel: 60}

	cases := []struct {
		name  string
		role  *models.Role
		actor MemberActor
		want  error
	}{
		{"super_admin never assignable", &models.Role{Name: "super_admin", HierarchyLevel: 100}, admin, ErrRoleNotAssignable},
		{"level 100 never assignable", &models.Role{Name: "custom", HierarchyLevel: 100}, MemberActor{RoleLevel: 100}, ErrRoleNotAssignable},
		{"nil role", nil, admin, ErrRoleNotAssignable},
		{"admin may grant admin", &models.Role{Name: "admin", HierarchyLevel: 80}, admin, nil},
		{"admin may grant user", &models.Role{Name: "user", HierarchyLevel: 40}, admin, nil},
		{"manager may not grant admin", &models.Role{Name: "admin", HierarchyLevel: 80}, manager, ErrInsufficientRoleLevel},
		{"manager may grant manager", &models.Role{Name: "manager", HierarchyLevel: 60}, manager, nil},
		{"no tenant role may not grant anything", &models.Role{Name: "guest", HierarchyLevel: 20}, MemberActor{}, ErrInsufficientRoleLevel},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateAssignableRole(tc.role, tc.actor); !errors.Is(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUpdateMemberRole_RejectsSelfChange(t *testing.T) {
	// The self-change rule is checked before any repository access.
	svc := NewTenantService(nil, nil)
	me := uuid.New()
	err := svc.UpdateMemberRole(context.Background(), uuid.New(), me, uuid.New(), MemberActor{UserID: me, RoleLevel: 80})
	if !errors.Is(err, ErrCannotChangeOwnRole) {
		t.Fatalf("got %v, want ErrCannotChangeOwnRole", err)
	}
}
