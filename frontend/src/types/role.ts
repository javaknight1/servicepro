export interface Permission {
  id: string;
  name: string;
  description: string;
  resource: string;
  action: string;
  created_at: string;
  updated_at: string;
}

export interface Role {
  id: string;
  name: string;
  description: string;
  hierarchy_level: number;
  parent_role_id?: string | null;
  permissions?: Permission[];
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
}

export interface RoleAssignment {
  user_id: string;
  role_id: string;
  assigned_at: string;
  assigned_by?: string | null;
  expires_at?: string | null;
}

export interface RoleAuditLog {
  id: string;
  role_id: string;
  role_name: string;
  action:
    | 'created'
    | 'updated'
    | 'deleted'
    | 'assigned'
    | 'unassigned'
    | 'permission_added'
    | 'permission_removed';
  performed_by: string;
  performed_by_name?: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  details?: Record<string, any>;
  timestamp: string;
}

export interface CreateRoleRequest {
  name: string;
  description: string;
  hierarchy_level: number;
  parent_role_id?: string | null;
  permission_ids?: string[];
}

export interface UpdateRoleRequest {
  name?: string;
  description?: string;
  hierarchy_level?: number;
  parent_role_id?: string | null;
  permission_ids?: string[];
}

export interface AssignRoleRequest {
  user_id: string;
  role_id: string;
  expires_at?: string | null;
}

export interface BulkAssignRolesRequest {
  user_ids: string[];
  role_id: string;
  expires_at?: string | null;
}

export interface RoleFilters {
  search?: string;
  hierarchy_level?: number;
  parent_role_id?: string;
  has_permission?: string;
}

export interface AuditFilters {
  role_id?: string;
  action?: string;
  performed_by?: string;
  start_date?: string;
  end_date?: string;
}
