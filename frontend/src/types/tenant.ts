// Tenant types for multi-tenancy support

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  owner_id: string;
  email?: string;
  phone?: string;
  logo_url?: string;
  settings?: TenantSettings;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  role?: string; // User's role in this tenant (when listing user's tenants)
}

export interface TenantSettings {
  isDefault?: boolean;
  timezone?: string;
  dateFormat?: string;
  currency?: string;
  invoicePrefix?: string;
  quotePrefix?: string;
  jobPrefix?: string;
  defaultPaymentTermId?: string;
  defaultTaxRateId?: string;
}

export interface TenantMember {
  id: string;
  user_id: string;
  email: string;
  first_name?: string;
  last_name?: string;
  profile_picture_url?: string;
  role_id: string;
  role_name: string;
  is_active: boolean;
  invited_at?: string;
  accepted_at?: string;
  joined_at: string;
}

// Request types
export interface CreateTenantRequest {
  name: string;
  slug: string;
  email?: string;
  phone?: string;
  settings?: TenantSettings;
}

export interface UpdateTenantRequest {
  name?: string;
  slug?: string;
  email?: string;
  phone?: string;
  logo_url?: string;
  settings?: TenantSettings;
  is_active?: boolean;
}

export interface AddMemberRequest {
  email: string;
  role_id: string;
}

export interface UpdateMemberRoleRequest {
  role_id: string;
}

// Response types
export interface TenantListResponse {
  tenants: Tenant[];
}

export interface TenantMembersResponse {
  members: TenantMember[];
}

// Store state
export interface TenantState {
  // Current tenant context
  currentTenant: Tenant | null;
  tenants: Tenant[];
  isLoading: boolean;
  error: string | null;

  // Actions
  loadTenants: () => Promise<void>;
  setCurrentTenant: (tenant: Tenant) => void;
  switchTenant: (tenantId: string) => Promise<void>;
  createTenant: (data: CreateTenantRequest) => Promise<Tenant>;
  updateTenant: (id: string, data: UpdateTenantRequest) => Promise<Tenant>;
  deleteTenant: (id: string) => Promise<void>;

  // Member management
  getMembers: (tenantId: string) => Promise<TenantMember[]>;
  addMember: (
    tenantId: string,
    data: AddMemberRequest
  ) => Promise<TenantMember>;
  removeMember: (tenantId: string, userId: string) => Promise<void>;
  updateMemberRole: (
    tenantId: string,
    userId: string,
    roleId: string
  ) => Promise<void>;

  // Utilities
  clearError: () => void;
  reset: () => void;
}
