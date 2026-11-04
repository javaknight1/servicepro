package permissions

import (
	"strings"
	"testing"
)

// Every permission a role or resource group mentions is defined, and every
// defined permission belongs to its resource group.
func TestPermissionTablesAreConsistent(t *testing.T) {
	defined := map[string]string{}
	for _, p := range AllPermissions() {
		defined[p.Name] = p.Resource
	}
	grouped := map[string]bool{}
	for resource, names := range PermissionsByResource() {
		for _, n := range names {
			if defined[n] != resource {
				t.Errorf("%s grouped under %s but defined under %q", n, resource, defined[n])
			}
			grouped[n] = true
		}
	}
	for n := range defined {
		if !grouped[n] {
			t.Errorf("%s missing from PermissionsByResource", n)
		}
	}
	for _, r := range DefaultRolePermissions() {
		for _, n := range r.Permissions {
			if _, ok := defined[n]; !ok {
				t.Errorf("role %s grants undefined %s", r.RoleName, n)
			}
		}
	}
}

// T035: technicians see their assigned jobs only and nothing about money,
// customers, reports or the team; every other built-in role keeps every job.
func TestTechnicianRolePermissions(t *testing.T) {
	roles := map[string][]string{}
	for _, r := range DefaultRolePermissions() {
		roles[r.RoleName] = r.Permissions
	}
	tech, ok := roles[RoleTechnician]
	if !ok {
		t.Fatal("no technician role")
	}
	for _, p := range tech {
		if !strings.HasPrefix(p, ResourceJobs+".") {
			t.Errorf("technician granted non-job permission %s", p)
		}
		if p == JobsViewAll || p == JobsCreate || p == JobsDelete || p == JobsAssign {
			t.Errorf("technician granted %s", p)
		}
	}
	for _, name := range []string{RoleSuperAdmin, RoleAdmin, RoleManager, RoleUser, RoleGuest} {
		found := false
		for _, p := range roles[name] {
			found = found || p == JobsViewAll
		}
		if !found {
			t.Errorf("%s lost %s", name, JobsViewAll)
		}
	}
}

// T192: integrations.manage (connect QuickBooks, which writes into the
// business's books) is for owners and admins only.
func TestIntegrationsManageOwnersAndAdminsOnly(t *testing.T) {
	for _, r := range DefaultRolePermissions() {
		has := false
		for _, p := range r.Permissions {
			has = has || p == IntegrationsManage
		}
		want := r.RoleName == RoleSuperAdmin || r.RoleName == RoleAdmin
		if has != want {
			t.Errorf("role %s: integrations.manage = %v, want %v", r.RoleName, has, want)
		}
	}
}
