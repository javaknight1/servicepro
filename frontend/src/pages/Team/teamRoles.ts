/**
 * The three roles tenants work with (T035, mockup 13): Owner (the built-in
 * `admin` role), Office manager (`manager`) and Technician (`technician`).
 * `user` and `guest` stay in the API but aren't offered; `super_admin` is
 * never assignable.
 */
import type { BadgeVariant } from '@components/shared';
import type { Role } from '@/types/role';

interface TeamRole {
  /** The API role name. */
  name: string;
  label: string;
  /** One line on what the role is for, shown under the picker. */
  summary: string;
}

const TEAM_ROLES: readonly TeamRole[] = [
  {
    name: 'admin',
    label: 'Owner',
    summary: 'Everything, including billing, plan, integrations and the team.',
  },
  {
    name: 'manager',
    label: 'Office manager',
    summary:
      'Schedule and dispatch, customers, quotes, invoices, payments and reports.',
  },
  {
    name: 'technician',
    label: 'Technician',
    summary:
      'Only the jobs assigned to them. Never sees prices, invoices or payments.',
  },
];

/** A role the team picker offers: the API role plus its label and summary. */
export interface TeamRoleOption {
  id: string;
  name: string;
  label: string;
  summary: string;
}

/** The offered roles, in Owner / Office manager / Technician order. */
export function teamRoleOptions(roles: Role[]): TeamRoleOption[] {
  return TEAM_ROLES.flatMap((teamRole) => {
    const role = roles.find((r) => r.name === teamRole.name);
    return role ? [{ ...teamRole, id: role.id }] : [];
  });
}

/** "Owner" for admin, "Office manager" for manager, … else the API name. */
export function roleLabel(roleName: string): string {
  const name = roleName.toLowerCase();
  return TEAM_ROLES.find((r) => r.name === name)?.label ?? roleName;
}

/** Role pill colours from mockup 13: owner orange, office teal, the rest grey. */
export function roleBadgeVariant(roleName: string): BadgeVariant {
  const name = roleName.toLowerCase();
  if (name === 'owner' || name === 'admin' || name === 'super_admin')
    return 'info';
  if (name === 'manager' || name.includes('office')) return 'teal';
  return 'neutral';
}
