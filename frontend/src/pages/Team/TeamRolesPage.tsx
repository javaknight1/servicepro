import { useState, useEffect } from 'react';
import { DashboardLayout } from '@components/layout';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  Badge,
} from '@components/shared';
import { useTenantStore } from '@store';
import { roleApi } from '@services/roleApi';
import { Shield, ChevronRight, Check, Building2, Info } from 'lucide-react';
import type { Role, Permission } from '@/types/role';

export function TeamRolesPage() {
  const { currentTenant } = useTenantStore();

  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // Selected role for viewing permissions
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [selectedPermissions, setSelectedPermissions] = useState<Set<string>>(
    new Set()
  );

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setIsLoading(true);
    try {
      const [roleList, permList] = await Promise.all([
        roleApi.getRoles().then((res) => res.data),
        roleApi.getPermissions().then((res) => res.data),
      ]);
      setRoles(roleList);
      setPermissions(permList);

      // Select first role by default
      if (roleList.length > 0 && !selectedRole) {
        selectRole(roleList[0]);
      }
    } catch (err) {
      console.error('Failed to load data:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const selectRole = async (role: Role) => {
    setSelectedRole(role);
    try {
      const rolePerms = await roleApi
        .getRolePermissions(role.id)
        .then((res) => res.data);
      setSelectedPermissions(new Set(rolePerms.map((p) => p.id)));
    } catch (err) {
      console.error('Failed to load role permissions:', err);
      setSelectedPermissions(new Set());
    }
  };

  // Group permissions by resource
  const permissionsByResource = permissions.reduce(
    (acc, perm) => {
      if (!acc[perm.resource]) {
        acc[perm.resource] = [];
      }
      acc[perm.resource].push(perm);
      return acc;
    },
    {} as Record<string, Permission[]>
  );

  if (!currentTenant) {
    return (
      <DashboardLayout>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <Card variant="elevated" padding="lg">
            <CardContent>
              <div className="text-center py-8">
                <Building2 className="h-12 w-12 text-neutral-400 mx-auto mb-4" />
                <h2 className="text-lg font-medium text-neutral-900 mb-2">
                  No Organization Selected
                </h2>
                <p className="text-neutral-600">
                  Please select an organization to view roles and permissions.
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-neutral-900">
            Roles & Permissions
          </h1>
          <p className="text-neutral-600 mt-2">
            View the available roles and their permissions
          </p>
        </div>

        {/* Info Banner */}
        <div className="mb-6 flex items-start gap-3 p-4 bg-blue-50 border border-blue-200 rounded-lg">
          <Info className="h-5 w-5 text-blue-600 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm text-blue-900 font-medium">System Roles</p>
            <p className="text-sm text-blue-700">
              These are the default roles available in your organization. Each
              role has a predefined set of permissions that determine what
              actions members can perform.
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Roles List */}
          <Card variant="elevated" padding="sm" className="lg:col-span-1">
            <CardHeader>
              <CardTitle className="text-lg">Roles</CardTitle>
              <CardDescription>
                Select a role to view its permissions
              </CardDescription>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <div className="py-4 text-center text-neutral-500">
                  Loading...
                </div>
              ) : roles.length === 0 ? (
                <div className="py-4 text-center text-neutral-500">
                  No roles found
                </div>
              ) : (
                <div className="space-y-1">
                  {roles.map((role) => (
                    <button
                      key={role.id}
                      onClick={() => selectRole(role)}
                      className={`w-full flex items-center justify-between px-3 py-3 rounded-lg text-left transition-colors ${
                        selectedRole?.id === role.id
                          ? 'bg-primary-50 text-primary-700 border border-primary-200'
                          : 'hover:bg-neutral-50 border border-transparent'
                      }`}
                    >
                      <div className="flex items-center">
                        <Shield
                          className={`h-5 w-5 mr-3 ${
                            selectedRole?.id === role.id
                              ? 'text-primary-600'
                              : 'text-neutral-400'
                          }`}
                        />
                        <div>
                          <p className="text-sm font-medium">{role.name}</p>
                          <p className="text-xs text-neutral-500">
                            {role.description}
                          </p>
                        </div>
                      </div>
                      <ChevronRight
                        className={`h-4 w-4 ${
                          selectedRole?.id === role.id
                            ? 'text-primary-600'
                            : 'text-neutral-400'
                        }`}
                      />
                    </button>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Permissions Panel */}
          <Card variant="elevated" padding="sm" className="lg:col-span-2">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-lg">
                    {selectedRole
                      ? `${selectedRole.name} Permissions`
                      : 'Select a Role'}
                  </CardTitle>
                  {selectedRole && (
                    <CardDescription>
                      {selectedPermissions.size} permissions granted
                    </CardDescription>
                  )}
                </div>
                {selectedRole && (
                  <Badge variant="default">
                    Level {selectedRole.hierarchy_level}
                  </Badge>
                )}
              </div>
            </CardHeader>
            <CardContent>
              {!selectedRole ? (
                <div className="py-8 text-center text-neutral-500">
                  <Shield className="h-12 w-12 text-neutral-300 mx-auto mb-3" />
                  <p>Select a role to view its permissions</p>
                </div>
              ) : isLoading ? (
                <div className="py-8 text-center text-neutral-500">
                  Loading permissions...
                </div>
              ) : (
                <div className="space-y-6">
                  {Object.entries(permissionsByResource).map(
                    ([resource, perms]) => (
                      <div key={resource}>
                        <h4 className="text-sm font-semibold text-neutral-900 uppercase tracking-wider mb-3 flex items-center">
                          <span className="bg-neutral-100 px-2 py-1 rounded">
                            {resource.replace(/_/g, ' ')}
                          </span>
                        </h4>
                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                          {perms.map((perm) => {
                            const isGranted = selectedPermissions.has(perm.id);
                            return (
                              <div
                                key={perm.id}
                                className={`flex items-center justify-between p-3 rounded-lg border ${
                                  isGranted
                                    ? 'border-green-200 bg-green-50'
                                    : 'border-neutral-200 bg-neutral-50 opacity-50'
                                }`}
                              >
                                <div className="text-left">
                                  <p
                                    className={`text-sm font-medium ${
                                      isGranted
                                        ? 'text-green-900'
                                        : 'text-neutral-500'
                                    }`}
                                  >
                                    {perm.action}
                                  </p>
                                  <p
                                    className={`text-xs ${
                                      isGranted
                                        ? 'text-green-700'
                                        : 'text-neutral-400'
                                    }`}
                                  >
                                    {perm.description}
                                  </p>
                                </div>
                                {isGranted && (
                                  <Check className="h-5 w-5 text-green-600 flex-shrink-0" />
                                )}
                              </div>
                            );
                          })}
                        </div>
                      </div>
                    )
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </DashboardLayout>
  );
}
