// Membership types for tenant-level membership system

/** Solo, Team and Pro are sold (D15); Free and Basic are legacy. */
export type MembershipTierName = 'solo' | 'team' | 'pro' | 'free' | 'basic';

export type MembershipStatus = 'active' | 'cancelled' | 'expired' | 'pending';

export type BillingCycle = 'monthly' | 'annual';

// Features available in each tier. The D15 tiers carry only the user limit;
// the count limits exist on the legacy tiers.
export interface TierFeatures {
  team_members: number; // -1 for unlimited
  customers?: number; // legacy; -1 for unlimited
  jobs_per_month?: number; // legacy
  quotes_per_month?: number; // legacy
  storage_gb?: number; // legacy
  advanced_reporting: boolean;
  api_access: boolean;
  custom_branding: boolean;
  multi_entity?: boolean;
  highlights?: string[]; // plan card bullets
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
  trial_ends_at?: string;
}

/** A no-card trial (D15). */
export interface TrialInfo {
  ends_at: string;
  days_left: number;
  ended: boolean;
}

// Combined membership response
export interface TenantMembership {
  subscription: TenantSubscription | null;
  tier: MembershipTier | null;
  trial?: TrialInfo;
  has_paid_subscription?: boolean;
  checkout_available?: boolean;
  founding_offer_available?: boolean;
}

// Request types
export interface UpdateMembershipRequest {
  tier_id: string;
  billing_cycle: BillingCycle;
  change_reason?: string;
}

export interface PlanCheckoutRequest {
  tier_id: string;
  billing_cycle: BillingCycle;
}

export interface PlanCheckoutResponse {
  url: string;
  founding_offer: boolean;
}

// Response types
export interface MembershipTiersResponse {
  tiers: MembershipTier[];
}

export interface SubscriptionHistoryResponse {
  history: TenantSubscription[];
}

// Subscription change preview
export interface PreviewSubscriptionChangeRequest {
  tier_id: string;
  billing_cycle: BillingCycle;
}

export interface SubscriptionChangePreview {
  // Current plan info
  current_tier_name: string;
  current_price_cents: number;
  current_billing_cycle: BillingCycle;
  current_period_end?: string;
  days_remaining: number;

  // New plan info
  new_tier_name: string;
  new_price_cents: number;
  new_billing_cycle: BillingCycle;

  // Proration info
  is_upgrade: boolean;
  is_downgrade: boolean;
  is_cancellation: boolean;
  proration_credit_cents: number;
  amount_due_today_cents: number;
  next_billing_date?: string;
  next_billing_amount_cents: number;

  // Warnings/messages
  warnings?: string[];
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
export function isUnlimited(value: number): boolean {
  return value === -1;
}

/** "1 user" / "Unlimited users" */
export function usersLabel(teamMembers: number): string {
  if (isUnlimited(teamMembers)) return 'Unlimited users';
  return `${teamMembers} ${teamMembers === 1 ? 'user' : 'users'}`;
}

/** Founding offer (D15): $40 off Team monthly, for life. */
export const FOUNDING_DISCOUNT_CENTS = 4000;
