// Membership types for tenant-level membership system

export type MembershipTierName = 'free' | 'basic' | 'pro';

export type MembershipStatus = 'active' | 'cancelled' | 'expired' | 'pending';

export type BillingCycle = 'monthly' | 'annual';

// Features available in each tier
export interface TierFeatures {
  team_members: number; // -1 for unlimited
  customers: number; // -1 for unlimited
  jobs_per_month: number; // -1 for unlimited
  quotes_per_month: number; // -1 for unlimited
  storage_gb: number;
  advanced_reporting: boolean;
  api_access: boolean;
  webhooks: boolean;
  custom_branding: boolean;
}

// Membership tier definition
export interface MembershipTier {
  id: string;
  name: MembershipTierName;
  display_name: string;
  description?: string;
  monthly_price_cents: number;
  annual_price_cents: number;
  features: TierFeatures;
  enabled: boolean; // false for legacy tiers (grandfathered users only)
  sort_order: number;
}

// Tenant subscription
export interface TenantSubscription {
  id: string;
  tenant_id: string;
  tier_id: string;
  tier_name: string;
  tier_display_name: string;
  status: MembershipStatus;
  billing_cycle: BillingCycle;
  price_cents: number; // Locked-in price at subscription time
  is_legacy: boolean; // True if on a disabled/legacy tier
  started_at: string;
  ended_at?: string;
  current_period_start: string;
  current_period_end?: string;
}

// Combined membership response
export interface TenantMembership {
  subscription: TenantSubscription | null;
  tier: MembershipTier | null;
}

// Request types
export interface UpdateMembershipRequest {
  tier_id: string;
  billing_cycle: BillingCycle;
  change_reason?: string;
}

// Response types
export interface MembershipTiersResponse {
  tiers: MembershipTier[];
}

export interface SubscriptionHistoryResponse {
  history: TenantSubscription[];
}

// Store state
export interface MembershipState {
  tiers: MembershipTier[];
  currentMembership: TenantMembership | null;
  isLoading: boolean;
  error: string | null;

  // Actions
  loadTiers: () => Promise<void>;
  loadMembership: (tenantId: string) => Promise<void>;
  updateMembership: (
    tenantId: string,
    tierId: string,
    billingCycle: BillingCycle
  ) => Promise<void>;
  clearError: () => void;
  reset: () => void;
}

// Helper functions
export function formatPrice(cents: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(cents / 100);
}

export function isUnlimited(value: number): boolean {
  return value === -1;
}

export function formatLimit(value: number): string {
  return isUnlimited(value) ? 'Unlimited' : value.toString();
}

// Feature display configuration
export interface FeatureDisplay {
  key: keyof TierFeatures;
  label: string;
  type: 'number' | 'boolean';
  unit?: string;
}

export const featureDisplayConfig: FeatureDisplay[] = [
  { key: 'team_members', label: 'Team Members', type: 'number' },
  { key: 'customers', label: 'Customers', type: 'number' },
  { key: 'jobs_per_month', label: 'Jobs / Month', type: 'number' },
  { key: 'quotes_per_month', label: 'Quotes / Month', type: 'number' },
  { key: 'storage_gb', label: 'Storage', type: 'number', unit: 'GB' },
  { key: 'advanced_reporting', label: 'Advanced Reporting', type: 'boolean' },
  { key: 'api_access', label: 'API Access', type: 'boolean' },
  { key: 'webhooks', label: 'Webhooks', type: 'boolean' },
  { key: 'custom_branding', label: 'Custom Branding', type: 'boolean' },
];
