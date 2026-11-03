/**
 * What the caller may do in an organization (tenant), for showing or hiding
 * actions.
 *
 * The API is the authority: every `/tenants/:id/*` route checks the caller's
 * permission in that tenant and answers 403 otherwise. The UI asks the API for
 * the caller's effective permission names in the tenant (`GET /tenants/:id/me`)
 * and gates actions on those, so custom roles get the right affordances.
 *
 * While that list is loading (or if it can't be loaded) a conservative
 * fallback applies: the management permissions follow the built-in role
 * defaults by role name (custom roles keep the action and rely on the 403),
 * and every other permission — notably `tenants.delete` — stays hidden.
 */
import { useQuery } from '@tanstack/react-query';
import { tenantApi } from '@services/tenantService';
import type { Tenant, TenantAccess } from '@/types/tenant';

/** Permission names the organization UI gates on. */
export const TENANT_PERMISSIONS = {
  /** Member and invitation management. */
  manageMembers: 'tenants.manage',
  /** Editing organization settings. */
  update: 'tenants.update',
  /** Deleting the organization. */
  delete: 'tenants.delete',
  /** Changing the plan. */
  membershipUpdate: 'membership.update',
  /** Payment methods and billing. */
  membershipBilling: 'membership.billing',
  /** Creating, editing and deactivating tax rates (T237). */
  manageTaxRates: 'tax_rates.manage',
} as const;

/**
 * Permission names the app's areas gate on (T035). Technicians hold only the
 * job ones (their assigned jobs); everything about money, customers, reports
 * and the team stays hidden from them.
 */
export const APP_PERMISSIONS = {
  customersList: 'customers.list',
  customersRead: 'customers.read',
  customersCreate: 'customers.create',
  /** Every job (office staff); without it the API returns assigned jobs only. */
  jobsViewAll: 'jobs.view_all',
  jobsList: 'jobs.list',
  jobsCreate: 'jobs.create',
  jobsDelete: 'jobs.delete',
  jobsAssign: 'jobs.assign',
  quotesList: 'quotes.list',
  quotesCreate: 'quotes.create',
  invoicesList: 'invoices.list',
  /** Seeing money at all: prices, costs, balances, billing rules. */
  invoicesRead: 'invoices.read',
  invoicesCreate: 'invoices.create',
  reportsView: 'reports.view',
  /** Team list and organization settings. */
  tenantsRead: 'tenants.read',
} as const;

/** Permissions the loading fallback may grant from the role name. */
const ROLE_FALLBACK_PERMISSIONS: ReadonlySet<string> = new Set([
  TENANT_PERMISSIONS.manageMembers,
  TENANT_PERMISSIONS.update,
  TENANT_PERMISSIONS.membershipUpdate,
  TENANT_PERMISSIONS.membershipBilling,
]);

/** Built-in roles whose defaults exclude the management permissions. */
const ROLES_WITHOUT_MANAGE: ReadonlySet<string> = new Set([
  'manager',
  'user',
  'guest',
  'technician',
]);

/**
 * Office permissions the loading fallback grants every role except
 * technicians, so office staff don't see their navigation flicker while the
 * list loads and technicians never see money or admin areas flash by.
 */
const OFFICE_FALLBACK_PERMISSIONS: ReadonlySet<string> = new Set(
  Object.values(APP_PERMISSIONS).filter((p) => p !== APP_PERMISSIONS.jobsList)
);

/** The built-in field role (T035): assigned jobs only, no money. */
const FIELD_ROLE = 'technician';

function normalizeRole(role: string | null | undefined): string | null {
  const value = role?.trim().toLowerCase();
  return value ? value : null;
}

/**
 * Whether the caller holds `permission`. With the API's list (`access`) this
 * is an exact lookup; without it, the role-name fallback described above.
 */
export function hasTenantPermission(
  permission: string,
  access: TenantAccess | null | undefined,
  role?: string | null
): boolean {
  if (access) return access.permissions.includes(permission);
  const name = normalizeRole(role);
  if (permission === APP_PERMISSIONS.jobsList) return true;
  if (OFFICE_FALLBACK_PERMISSIONS.has(permission)) return name !== FIELD_ROLE;
  if (!ROLE_FALLBACK_PERMISSIONS.has(permission)) return false;
  return name === null || !ROLES_WITHOUT_MANAGE.has(name);
}

/** React Query key for the caller's access in one tenant. */
export const tenantAccessQueryKey = (tenantId: string | null | undefined) =>
  ['tenant-access', tenantId ?? null] as const;

export interface TenantPermissions {
  /** Whether the caller holds `permission` in this tenant. */
  can: (permission: string) => boolean;
  /** tenants.manage: members and invitations. */
  canManageMembers: boolean;
  /** tenants.update: organization settings. */
  canUpdateTenant: boolean;
  /** membership.update: plan changes. */
  canChangePlan: boolean;
  /** membership.billing: payment methods and billing. */
  canManageBilling: boolean;
  /** tenants.delete: deleting the organization. */
  canDeleteTenant: boolean;
  /** tax_rates.manage: adding and changing tax rates. */
  canManageTaxRates: boolean;
  /** True until the API's permission list for this tenant has loaded. */
  isLoading: boolean;
}

/**
 * The caller's permissions in `tenant`, fetched from `GET /tenants/:id/me`
 * and cached per tenant.
 */
export function useTenantPermissions(
  tenant: Pick<Tenant, 'id' | 'role'> | null | undefined
): TenantPermissions {
  const tenantId = tenant?.id;
  const { data, isLoading } = useQuery({
    queryKey: tenantAccessQueryKey(tenantId),
    queryFn: async () => {
      const response = await tenantApi.getMyAccess(tenantId as string);
      return response.data;
    },
    enabled: !!tenantId,
    staleTime: 60_000,
  });

  // Ignore a cached answer for another tenant (defensive; the key is per tenant).
  const access = data && data.tenant_id === tenantId ? data : undefined;
  const can = (permission: string) =>
    hasTenantPermission(permission, access, tenant?.role);

  return {
    can,
    canManageMembers: can(TENANT_PERMISSIONS.manageMembers),
    canUpdateTenant: can(TENANT_PERMISSIONS.update),
    canChangePlan: can(TENANT_PERMISSIONS.membershipUpdate),
    canManageBilling: can(TENANT_PERMISSIONS.membershipBilling),
    canDeleteTenant: can(TENANT_PERMISSIONS.delete),
    canManageTaxRates: can(TENANT_PERMISSIONS.manageTaxRates),
    isLoading: !!tenantId && isLoading,
  };
}
