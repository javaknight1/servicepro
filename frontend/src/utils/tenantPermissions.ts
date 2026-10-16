/**
 * Client-side hints for what the caller may do in the current organization.
 *
 * The API is the authority: every `/tenants/:id/*` route checks the caller's
 * permission in that tenant and answers 403 otherwise. The only signal the
 * client has is the caller's role *name* in the tenant (`Tenant.role`, from
 * `GET /tenants`), so these helpers only hide actions for the built-in roles
 * whose default permissions are known to exclude them
 * (backend/internal/permissions/constants.go DefaultRolePermissions).
 * Unknown or custom roles get the action and rely on the 403 handling.
 */

/** Built-in roles without tenants.manage / tenants.update / membership.update. */
const ROLES_WITHOUT_MANAGE = new Set(['manager', 'user', 'guest']);

/** Built-in roles without tenants.delete (only super_admin has it). */
const ROLES_WITHOUT_DELETE = new Set(['admin', 'manager', 'user', 'guest']);

function normalize(role: string | null | undefined): string | null {
  const value = role?.trim().toLowerCase();
  return value ? value : null;
}

/**
 * Whether to offer member/invitation management, organization settings edits
 * and plan changes (tenants.manage, tenants.update, membership.update).
 */
export function canManageTenant(role: string | null | undefined): boolean {
  const name = normalize(role);
  return name === null || !ROLES_WITHOUT_MANAGE.has(name);
}

/** Whether to offer deleting the organization (tenants.delete). */
export function canDeleteTenant(role: string | null | undefined): boolean {
  const name = normalize(role);
  return name === null || !ROLES_WITHOUT_DELETE.has(name);
}
