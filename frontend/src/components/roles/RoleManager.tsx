import { useState } from 'react';
import { Plus, Search, Trash2, Edit2, Shield, Users } from 'lucide-react';
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  Modal,
} from '@components/shared';
import { useRoles, useDeleteRole, useBulkAssignRoles } from '@hooks/useRoles';
import { RoleForm } from './RoleForm';
import type { Role } from '@types/role';

interface RoleManagerProps {
  onRoleSelect?: (role: Role) => void;
}

export function RoleManager({ onRoleSelect }: RoleManagerProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedRoles, setSelectedRoles] = useState<string[]>([]);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [deletingRole, setDeletingRole] = useState<Role | null>(null);

  // Queries
  const { data: roles = [], isLoading, error } = useRoles();

  // Mutations
  const deleteRoleMutation = useDeleteRole();
  const bulkAssignMutation = useBulkAssignRoles();

  // Filtered roles based on search
  const filteredRoles = roles.filter(
    (role) =>
      role.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      role.description.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Handle role selection (checkbox)
  const handleSelectRole = (roleId: string) => {
    setSelectedRoles((prev) =>
      prev.includes(roleId)
        ? prev.filter((id) => id !== roleId)
        : [...prev, roleId]
    );
  };

  // Handle select all
  const handleSelectAll = () => {
    if (selectedRoles.length === filteredRoles.length) {
      setSelectedRoles([]);
    } else {
      setSelectedRoles(filteredRoles.map((r) => r.id));
    }
  };

  // Handle delete role
  const handleDeleteRole = async () => {
    if (!deletingRole) return;

    try {
      await deleteRoleMutation.mutateAsync(deletingRole.id);
      setDeletingRole(null);
    } catch (error) {
      console.error('Failed to delete role:', error);
    }
  };

  // Handle bulk delete
  const handleBulkDelete = async () => {
    if (selectedRoles.length === 0) return;

    try {
      await Promise.all(
        selectedRoles.map((roleId) => deleteRoleMutation.mutateAsync(roleId))
      );
      setSelectedRoles([]);
    } catch (error) {
      console.error('Failed to bulk delete roles:', error);
    }
  };

  // Get role level badge color
  const getLevelBadgeColor = (level: number) => {
    if (level >= 80) return 'bg-error-100 text-error-700';
    if (level >= 50) return 'bg-warning-100 text-warning-700';
    return 'bg-neutral-100 text-neutral-700';
  };

  if (error) {
    return (
      <Card variant="outlined">
        <CardContent>
          <div className="text-center py-8">
            <p className="text-error-600">
              Failed to load roles. Please try again.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-neutral-900">
            Role Management
          </h2>
          <p className="text-sm text-neutral-600 mt-1">
            Manage roles and permissions for your organization
          </p>
        </div>
        <Button onClick={() => setIsCreateModalOpen(true)} size="md">
          <Plus className="h-4 w-4 mr-2" />
          Create Role
        </Button>
      </div>

      {/* Search and Bulk Actions */}
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-neutral-400" />
              <Input
                type="text"
                placeholder="Search roles..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
                fullWidth
              />
            </div>
            {selectedRoles.length > 0 && (
              <div className="flex items-center gap-2">
                <span className="text-sm text-neutral-600">
                  {selectedRoles.length} selected
                </span>
                <Button
                  variant="danger"
                  size="sm"
                  onClick={handleBulkDelete}
                  isLoading={deleteRoleMutation.isPending}
                >
                  <Trash2 className="h-4 w-4 mr-1" />
                  Delete
                </Button>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Roles Table */}
      <Card>
        <CardHeader>
          <CardTitle>Roles</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="text-center py-8">
              <p className="text-neutral-600">Loading roles...</p>
            </div>
          ) : filteredRoles.length === 0 ? (
            <div className="text-center py-8">
              <Shield className="h-12 w-12 text-neutral-400 mx-auto mb-4" />
              <p className="text-neutral-600">No roles found</p>
              {searchQuery && (
                <p className="text-sm text-neutral-500 mt-1">
                  Try adjusting your search
                </p>
              )}
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-neutral-200">
                <thead className="bg-neutral-50">
                  <tr>
                    <th className="px-4 py-3 text-left">
                      <input
                        type="checkbox"
                        checked={selectedRoles.length === filteredRoles.length}
                        onChange={handleSelectAll}
                        className="rounded border-neutral-300"
                      />
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-neutral-500 uppercase tracking-wider">
                      Name
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-neutral-500 uppercase tracking-wider">
                      Description
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-neutral-500 uppercase tracking-wider">
                      Level
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-neutral-500 uppercase tracking-wider">
                      Permissions
                    </th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-neutral-500 uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-neutral-200">
                  {filteredRoles.map((role) => (
                    <tr
                      key={role.id}
                      className="hover:bg-neutral-50 cursor-pointer"
                      onClick={() => onRoleSelect?.(role)}
                    >
                      <td
                        className="px-4 py-4"
                        onClick={(e) => e.stopPropagation()}
                      >
                        <input
                          type="checkbox"
                          checked={selectedRoles.includes(role.id)}
                          onChange={() => handleSelectRole(role.id)}
                          className="rounded border-neutral-300"
                        />
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap">
                        <div className="flex items-center">
                          <Shield className="h-5 w-5 text-primary-600 mr-2" />
                          <span className="text-sm font-medium text-neutral-900">
                            {role.name}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-4">
                        <span className="text-sm text-neutral-600">
                          {role.description || 'No description'}
                        </span>
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap">
                        <span
                          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getLevelBadgeColor(
                            role.hierarchy_level
                          )}`}
                        >
                          Level {role.hierarchy_level}
                        </span>
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap text-sm text-neutral-600">
                        {role.permissions?.length || 0} permissions
                      </td>
                      <td className="px-4 py-4 whitespace-nowrap text-right text-sm font-medium">
                        <div
                          className="flex items-center justify-end gap-2"
                          onClick={(e) => e.stopPropagation()}
                        >
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setEditingRole(role)}
                          >
                            <Edit2 className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setDeletingRole(role)}
                            disabled={role.hierarchy_level >= 80}
                          >
                            <Trash2 className="h-4 w-4 text-error-600" />
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create Role Modal */}
      <Modal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        title="Create New Role"
      >
        <RoleForm
          onSuccess={() => setIsCreateModalOpen(false)}
          onCancel={() => setIsCreateModalOpen(false)}
        />
      </Modal>

      {/* Edit Role Modal */}
      <Modal
        isOpen={!!editingRole}
        onClose={() => setEditingRole(null)}
        title="Edit Role"
      >
        {editingRole && (
          <RoleForm
            role={editingRole}
            onSuccess={() => setEditingRole(null)}
            onCancel={() => setEditingRole(null)}
          />
        )}
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={!!deletingRole}
        onClose={() => setDeletingRole(null)}
        title="Delete Role"
      >
        <div className="space-y-4">
          <p className="text-neutral-600">
            Are you sure you want to delete the role{' '}
            <strong>{deletingRole?.name}</strong>? This action cannot be undone.
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={() => setDeletingRole(null)}>
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={handleDeleteRole}
              isLoading={deleteRoleMutation.isPending}
            >
              Delete Role
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
