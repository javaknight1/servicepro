import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button, Input } from '@components/shared';
import {
  useCreateRole,
  useUpdateRole,
  usePermissions,
  useRoles,
} from '@hooks/useRoles';
import { Loader2, Check } from 'lucide-react';
import type { Role, Permission } from '@app-types/role';

const roleSchema = z.object({
  name: z
    .string()
    .min(1, 'Role name is required')
    .max(100, 'Role name must be 100 characters or less')
    .regex(
      /^[a-zA-Z0-9_-]+$/,
      'Role name can only contain letters, numbers, underscores, and hyphens'
    ),
  description: z
    .string()
    .max(500, 'Description must be 500 characters or less')
    .optional(),
  hierarchy_level: z
    .number()
    .min(0, 'Hierarchy level must be at least 0')
    .max(100, 'Hierarchy level must be at most 100'),
  parent_role_id: z.string().nullable().optional(),
});

type RoleFormData = z.infer<typeof roleSchema>;

interface RoleFormProps {
  role?: Role;
  onSuccess?: () => void;
  onCancel?: () => void;
}

export function RoleForm({ role, onSuccess, onCancel }: RoleFormProps) {
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([]);
  const [permissionSearch, setPermissionSearch] = useState('');

  // Queries
  const { data: roles = [] } = useRoles();
  const { data: permissions = [], isLoading: loadingPermissions } =
    usePermissions();

  // Mutations
  const createMutation = useCreateRole();
  const updateMutation = useUpdateRole();

  // Form
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    reset: _reset,
  } = useForm<RoleFormData>({
    resolver: zodResolver(roleSchema),
    defaultValues: {
      name: role?.name || '',
      description: role?.description || '',
      hierarchy_level: role?.hierarchy_level || 0,
      parent_role_id: role?.parent_role_id || null,
    },
  });

  // Initialize selected permissions
  useEffect(() => {
    if (role?.permissions) {
      setSelectedPermissions(role.permissions.map((p) => p.id));
    }
  }, [role]);

  // Handle permission toggle
  const handlePermissionToggle = (permissionId: string) => {
    setSelectedPermissions((prev) =>
      prev.includes(permissionId)
        ? prev.filter((id) => id !== permissionId)
        : [...prev, permissionId]
    );
  };

  // Handle select all permissions
  const handleSelectAllPermissions = () => {
    if (selectedPermissions.length === permissions.length) {
      setSelectedPermissions([]);
    } else {
      setSelectedPermissions(permissions.map((p) => p.id));
    }
  };

  // Filter permissions by search
  const filteredPermissions = permissions.filter(
    (permission) =>
      permission.name.toLowerCase().includes(permissionSearch.toLowerCase()) ||
      permission.resource
        .toLowerCase()
        .includes(permissionSearch.toLowerCase()) ||
      permission.action.toLowerCase().includes(permissionSearch.toLowerCase())
  );

  // Group permissions by resource
  const groupedPermissions = filteredPermissions.reduce(
    (acc, permission) => {
      if (!acc[permission.resource]) {
        acc[permission.resource] = [];
      }
      acc[permission.resource].push(permission);
      return acc;
    },
    {} as Record<string, Permission[]>
  );

  // Handle form submission
  const onSubmit = async (data: RoleFormData) => {
    try {
      if (role) {
        // Update existing role
        await updateMutation.mutateAsync({
          id: role.id,
          data: {
            ...data,
            permission_ids: selectedPermissions,
          },
        });
      } else {
        // Create new role
        await createMutation.mutateAsync({
          ...data,
          description: data.description || '',
          permission_ids: selectedPermissions,
        });
      }
      onSuccess?.();
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to save role:', error);
    }
  };

  const isLoading =
    isSubmitting || createMutation.isPending || updateMutation.isPending;

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      {/* Basic Information */}
      <div className="space-y-4">
        <h3 className="text-lg font-medium text-neutral-900">
          Basic Information
        </h3>

        <Input
          label="Role Name"
          placeholder="e.g., project_manager"
          error={errors.name?.message}
          fullWidth
          {...register('name')}
        />

        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">
            Description
          </label>
          <textarea
            placeholder="Describe this role's responsibilities..."
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
            rows={3}
            {...register('description')}
          />
          {errors.description && (
            <p className="text-sm text-error-600 mt-1">
              {errors.description.message}
            </p>
          )}
        </div>

        <div className="grid grid-cols-2 gap-4">
          <Input
            label="Hierarchy Level"
            type="number"
            min={0}
            max={100}
            helperText="0-100 (higher = more authority)"
            error={errors.hierarchy_level?.message}
            fullWidth
            {...register('hierarchy_level', { valueAsNumber: true })}
          />

          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-1">
              Parent Role
            </label>
            <select
              className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
              {...register('parent_role_id')}
            >
              <option value="">None</option>
              {roles
                .filter((r) => r.id !== role?.id) // Don't show current role
                .map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.name} (Level {r.hierarchy_level})
                  </option>
                ))}
            </select>
          </div>
        </div>
      </div>

      {/* Permissions */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-medium text-neutral-900">Permissions</h3>
          <button
            type="button"
            onClick={handleSelectAllPermissions}
            className="text-sm text-primary-600 hover:text-primary-700"
          >
            {selectedPermissions.length === permissions.length
              ? 'Deselect All'
              : 'Select All'}
          </button>
        </div>

        <Input
          type="text"
          placeholder="Search permissions..."
          value={permissionSearch}
          onChange={(e) => setPermissionSearch(e.target.value)}
          fullWidth
        />

        <div className="border border-neutral-200 rounded-lg max-h-96 overflow-y-auto">
          {loadingPermissions ? (
            <div className="p-8 text-center">
              <Loader2 className="h-6 w-6 animate-spin mx-auto text-primary-600" />
              <p className="text-sm text-neutral-600 mt-2">
                Loading permissions...
              </p>
            </div>
          ) : Object.keys(groupedPermissions).length === 0 ? (
            <div className="p-8 text-center">
              <p className="text-sm text-neutral-600">No permissions found</p>
            </div>
          ) : (
            <div className="divide-y divide-neutral-200">
              {Object.entries(groupedPermissions).map(([resource, perms]) => (
                <div key={resource} className="p-4">
                  <h4 className="text-sm font-medium text-neutral-900 mb-3 capitalize">
                    {resource}
                  </h4>
                  <div className="space-y-2">
                    {perms.map((permission) => (
                      <label
                        key={permission.id}
                        className="flex items-start space-x-3 cursor-pointer hover:bg-neutral-50 p-2 rounded"
                      >
                        <div className="relative flex items-center justify-center w-5 h-5 mt-0.5">
                          <input
                            type="checkbox"
                            checked={selectedPermissions.includes(
                              permission.id
                            )}
                            onChange={() =>
                              handlePermissionToggle(permission.id)
                            }
                            className="sr-only"
                          />
                          <div
                            className={`w-5 h-5 border-2 rounded flex items-center justify-center transition-colors ${
                              selectedPermissions.includes(permission.id)
                                ? 'bg-primary-600 border-primary-600'
                                : 'border-neutral-300 bg-white'
                            }`}
                          >
                            {selectedPermissions.includes(permission.id) && (
                              <Check className="h-3 w-3 text-white" />
                            )}
                          </div>
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-neutral-900">
                            {permission.name}
                          </p>
                          {permission.description && (
                            <p className="text-xs text-neutral-500 mt-0.5">
                              {permission.description}
                            </p>
                          )}
                          <p className="text-xs text-neutral-400 mt-0.5">
                            {permission.resource}.{permission.action}
                          </p>
                        </div>
                      </label>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="text-sm text-neutral-600">
          {selectedPermissions.length} permission
          {selectedPermissions.length !== 1 ? 's' : ''} selected
        </div>
      </div>

      {/* Actions */}
      <div className="flex justify-end gap-2 pt-4 border-t border-neutral-200">
        <Button
          type="button"
          variant="outline"
          onClick={onCancel}
          disabled={isLoading}
        >
          Cancel
        </Button>
        <Button type="submit" isLoading={isLoading}>
          {role ? 'Update Role' : 'Create Role'}
        </Button>
      </div>

      {/* Error Display */}
      {(createMutation.isError || updateMutation.isError) && (
        <div className="bg-error-50 border border-error-200 rounded-lg p-4">
          <p className="text-sm text-error-700">
            Failed to {role ? 'update' : 'create'} role. Please try again.
          </p>
        </div>
      )}
    </form>
  );
}
