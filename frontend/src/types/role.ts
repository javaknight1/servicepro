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

export interface RoleFilters {
  search?: string;
  hierarchy_level?: number;
  parent_role_id?: string;
  has_permission?: string;
}
