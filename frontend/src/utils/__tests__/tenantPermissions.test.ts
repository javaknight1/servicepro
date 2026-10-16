import { describe, it, expect } from 'vitest';
import { canDeleteTenant, canManageTenant } from '../tenantPermissions';

describe('tenantPermissions', () => {
  it('canManageTenant hides management only for built-in non-admin roles', () => {
    expect(canManageTenant('super_admin')).toBe(true);
    expect(canManageTenant('admin')).toBe(true);
    expect(canManageTenant('Admin')).toBe(true);
    expect(canManageTenant('manager')).toBe(false);
    expect(canManageTenant('user')).toBe(false);
    expect(canManageTenant('guest')).toBe(false);
  });

  it('canManageTenant defers to the API for unknown or custom roles', () => {
    expect(canManageTenant(undefined)).toBe(true);
    expect(canManageTenant(null)).toBe(true);
    expect(canManageTenant('')).toBe(true);
    expect(canManageTenant('office_lead')).toBe(true);
  });

  it('canDeleteTenant is only offered to super_admin among built-in roles', () => {
    expect(canDeleteTenant('super_admin')).toBe(true);
    expect(canDeleteTenant('admin')).toBe(false);
    expect(canDeleteTenant('user')).toBe(false);
    expect(canDeleteTenant(undefined)).toBe(true);
  });
});
