import { useState } from 'react';
import { format } from 'date-fns';
import {
  Download,
  Filter,
  Search,
  Calendar,
  User,
  Shield,
  Activity,
} from 'lucide-react';
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from '@components/shared';
import { useRoleAuditLogs, useExportAuditLogs } from '@hooks/useRoles';
import type { RoleAuditLog, AuditFilters } from '@types/role';

const actionLabels: Record<RoleAuditLog['action'], string> = {
  created: 'Created',
  updated: 'Updated',
  deleted: 'Deleted',
  assigned: 'Assigned',
  unassigned: 'Unassigned',
  permission_added: 'Permission Added',
  permission_removed: 'Permission Removed',
};

const actionColors: Record<RoleAuditLog['action'], string> = {
  created: 'bg-success-100 text-success-700',
  updated: 'bg-primary-100 text-primary-700',
  deleted: 'bg-error-100 text-error-700',
  assigned: 'bg-secondary-100 text-secondary-700',
  unassigned: 'bg-warning-100 text-warning-700',
  permission_added: 'bg-success-100 text-success-700',
  permission_removed: 'bg-error-100 text-error-700',
};

export function RoleAudit() {
  const [filters, setFilters] = useState<AuditFilters>({});
  const [showFilters, setShowFilters] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  // Queries
  const { data: auditLogs = [], isLoading, error } = useRoleAuditLogs(filters);
  const exportMutation = useExportAuditLogs();

  // Filter audit logs by search query
  const filteredLogs = auditLogs.filter(
    (log) =>
      log.role_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (log.performed_by_name?.toLowerCase() || '').includes(
        searchQuery.toLowerCase()
      ) ||
      log.action.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Handle filter change
  const handleFilterChange = (key: keyof AuditFilters, value: string) => {
    setFilters((prev) => ({
      ...prev,
      [key]: value || undefined,
    }));
  };

  // Handle clear filters
  const handleClearFilters = () => {
    setFilters({});
    setSearchQuery('');
  };

  // Handle export
  const handleExport = async () => {
    try {
      await exportMutation.mutateAsync(filters);
    } catch (error) {
      console.error('Failed to export audit logs:', error);
    }
  };

  // Get icon for action
  const getActionIcon = (action: RoleAuditLog['action']) => {
    switch (action) {
      case 'created':
      case 'deleted':
      case 'updated':
        return <Shield className="h-4 w-4" />;
      case 'assigned':
      case 'unassigned':
        return <User className="h-4 w-4" />;
      case 'permission_added':
      case 'permission_removed':
        return <Activity className="h-4 w-4" />;
      default:
        return <Shield className="h-4 w-4" />;
    }
  };

  if (error) {
    return (
      <Card variant="outlined">
        <CardContent>
          <div className="text-center py-8">
            <p className="text-error-600">
              Failed to load audit logs. Please try again.
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
            Role Audit Trail
          </h2>
          <p className="text-sm text-neutral-600 mt-1">
            Track all role and permission changes
          </p>
        </div>
        <Button
          onClick={handleExport}
          isLoading={exportMutation.isPending}
          size="md"
          variant="outline"
        >
          <Download className="h-4 w-4 mr-2" />
          Export CSV
        </Button>
      </div>

      {/* Search and Filters */}
      <Card>
        <CardContent className="p-4 space-y-4">
          <div className="flex items-center gap-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-neutral-400" />
              <Input
                type="text"
                placeholder="Search audit logs..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
                fullWidth
              />
            </div>
            <Button
              variant="outline"
              size="md"
              onClick={() => setShowFilters(!showFilters)}
            >
              <Filter className="h-4 w-4 mr-2" />
              Filters
              {Object.keys(filters).length > 0 && (
                <span className="ml-2 bg-primary-100 text-primary-700 rounded-full px-2 py-0.5 text-xs">
                  {Object.keys(filters).length}
                </span>
              )}
            </Button>
          </div>

          {/* Filter Panel */}
          {showFilters && (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 pt-4 border-t border-neutral-200">
              <div>
                <label className="block text-sm font-medium text-neutral-700 mb-1">
                  Action
                </label>
                <select
                  value={filters.action || ''}
                  onChange={(e) => handleFilterChange('action', e.target.value)}
                  className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                >
                  <option value="">All Actions</option>
                  <option value="created">Created</option>
                  <option value="updated">Updated</option>
                  <option value="deleted">Deleted</option>
                  <option value="assigned">Assigned</option>
                  <option value="unassigned">Unassigned</option>
                  <option value="permission_added">Permission Added</option>
                  <option value="permission_removed">Permission Removed</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-neutral-700 mb-1">
                  Start Date
                </label>
                <Input
                  type="date"
                  value={filters.start_date || ''}
                  onChange={(e) =>
                    handleFilterChange('start_date', e.target.value)
                  }
                  fullWidth
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-neutral-700 mb-1">
                  End Date
                </label>
                <Input
                  type="date"
                  value={filters.end_date || ''}
                  onChange={(e) =>
                    handleFilterChange('end_date', e.target.value)
                  }
                  fullWidth
                />
              </div>

              <div className="flex items-end">
                <Button
                  variant="outline"
                  size="md"
                  onClick={handleClearFilters}
                  fullWidth
                >
                  Clear Filters
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Audit Log List */}
      <Card>
        <CardHeader>
          <CardTitle>Activity Log</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="text-center py-8">
              <p className="text-neutral-600">Loading audit logs...</p>
            </div>
          ) : filteredLogs.length === 0 ? (
            <div className="text-center py-8">
              <Activity className="h-12 w-12 text-neutral-400 mx-auto mb-4" />
              <p className="text-neutral-600">No audit logs found</p>
              {(searchQuery || Object.keys(filters).length > 0) && (
                <p className="text-sm text-neutral-500 mt-1">
                  Try adjusting your search or filters
                </p>
              )}
            </div>
          ) : (
            <div className="space-y-4">
              {filteredLogs.map((log) => (
                <div
                  key={log.id}
                  className="flex items-start gap-4 p-4 border border-neutral-200 rounded-lg hover:bg-neutral-50 transition-colors"
                >
                  {/* Icon */}
                  <div
                    className={`flex items-center justify-center w-10 h-10 rounded-full ${
                      actionColors[log.action]
                    }`}
                  >
                    {getActionIcon(log.action)}
                  </div>

                  {/* Content */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span
                        className={`inline-flex items-center px-2 py-1 rounded-md text-xs font-medium ${
                          actionColors[log.action]
                        }`}
                      >
                        {actionLabels[log.action]}
                      </span>
                      <span className="text-sm font-medium text-neutral-900">
                        {log.role_name}
                      </span>
                    </div>

                    <div className="flex items-center gap-4 mt-2 text-xs text-neutral-500">
                      <span className="flex items-center gap-1">
                        <User className="h-3 w-3" />
                        {log.performed_by_name || 'System'}
                      </span>
                      <span className="flex items-center gap-1">
                        <Calendar className="h-3 w-3" />
                        {format(new Date(log.timestamp), 'MMM dd, yyyy HH:mm')}
                      </span>
                    </div>

                    {/* Details */}
                    {log.details && Object.keys(log.details).length > 0 && (
                      <div className="mt-2 p-2 bg-neutral-50 rounded text-xs">
                        <details>
                          <summary className="cursor-pointer text-neutral-600 hover:text-neutral-900">
                            View Details
                          </summary>
                          <pre className="mt-2 text-neutral-700 whitespace-pre-wrap">
                            {JSON.stringify(log.details, null, 2)}
                          </pre>
                        </details>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
