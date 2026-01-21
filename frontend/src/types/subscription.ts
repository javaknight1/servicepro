// =============================================================================
// Subscription Types
// =============================================================================

export type BillingCycle = 'monthly' | 'quarterly' | 'annual';

export type PlanTier =
  | 'free'
  | 'starter'
  | 'professional'
  | 'business'
  | 'enterprise';

export type SubscriptionStatus =
  | 'trialing'
  | 'active'
  | 'past_due'
  | 'canceled'
  | 'unpaid'
  | 'incomplete'
  | 'incomplete_expired'
  | 'paused';

export type CancellationReason =
  | 'user_requested'
  | 'payment_failed'
  | 'fraud'
  | 'admin'
  | 'other';

export type FeatureType = 'boolean' | 'numeric' | 'text';

// =============================================================================
// Plan Models
// =============================================================================

export interface PlanFeature {
  id: string;
  planId: string;
  featureKey: string;
  featureName: string;
  featureType: FeatureType;
  value?: string;
  numericValue?: number;
  isEnabled: boolean;
  description?: string;
  sortOrder: number;
}

export interface Plan {
  id: string;
  name: string;
  description: string;
  tier: PlanTier;
  monthlyPrice: number;
  annualPrice: number;
  quarterlyPrice: number;
  currency: string;
  trialDays: number;
  isActive: boolean;
  isPublic: boolean;
  sortOrder: number;
  features: PlanFeature[];
}

// =============================================================================
// Subscription Models
// =============================================================================

export interface Subscription {
  id: string;
  customerId: string;
  plan: Plan;
  status: SubscriptionStatus;
  billingCycle: BillingCycle;
  currentPeriodStart: string;
  currentPeriodEnd: string;
  trialEnd?: string;
  cancelAtPeriodEnd: boolean;
  cancellationReason?: CancellationReason;
  cancellationFeedback?: string;
  daysUntilRenewal: number;
  trialDaysRemaining: number;
  createdAt: string;
}

// =============================================================================
// Billing Models
// =============================================================================

export interface BillingPeriod {
  startDate: string;
  endDate: string;
  amount: number;
  isCurrent: boolean;
  isPaid: boolean;
}

export interface BillingSchedule {
  currentPeriod: BillingPeriod;
  upcomingPeriods: BillingPeriod[];
  billingCycle: BillingCycle;
  nextBillingDate: string;
  amountDue: number;
  currency: string;
}

export interface Invoice {
  id: string;
  subscriptionId: string;
  customerId: string;
  stripeInvoiceId: string;
  subtotal: number;
  tax: number;
  total: number;
  amountDue: number;
  amountPaid: number;
  status: 'draft' | 'open' | 'paid' | 'uncollectible' | 'void';
  periodStart: string;
  periodEnd: string;
  dueDate?: string;
  paidAt?: string;
  hostedInvoiceUrl?: string;
  invoicePdf?: string;
  lineItems: InvoiceLineItem[];
  createdAt: string;
}

export interface InvoiceLineItem {
  id: string;
  description: string;
  quantity: number;
  unitAmount: number;
  amount: number;
  proration: boolean;
}

export interface ProrationResult {
  creditAmount: number;
  chargeAmount: number;
  netAmount: number;
  daysRemaining: number;
  totalDays: number;
  effectiveDate: string;
  description: string;
}

export interface PlanChangePreview {
  currentPlan: Plan;
  newPlan: Plan;
  immediateCharge: number;
  nextBillingDate: string;
  nextBillingAmount: number;
  prorationDetails?: ProrationResult;
}

// =============================================================================
// Component Props
// =============================================================================

export interface SubscriptionManagerProps {
  subscription?: Subscription;
  plans: Plan[];
  onChangePlan?: (planId: string) => void;
  onChangeBillingCycle?: (cycle: BillingCycle) => void;
  onCancel?: () => void;
  onReactivate?: () => void;
  isLoading?: boolean;
  error?: string;
  className?: string;
}

export interface PlanSelectorProps {
  plans: Plan[];
  currentPlanId?: string;
  selectedPlanId?: string;
  billingCycle: BillingCycle;
  onSelectPlan?: (planId: string) => void;
  onChangeBillingCycle?: (cycle: BillingCycle) => void;
  showComparison?: boolean;
  highlightedPlanId?: string;
  disabled?: boolean;
  className?: string;
}

export interface BillingScheduleProps {
  schedule: BillingSchedule;
  invoices?: Invoice[];
  onViewInvoice?: (invoice: Invoice) => void;
  onDownloadInvoice?: (invoice: Invoice) => void;
  showPastInvoices?: boolean;
  maxInvoices?: number;
  isLoading?: boolean;
  className?: string;
}

export interface CancellationFlowProps {
  subscription: Subscription;
  onCancel: (
    reason: CancellationReason,
    feedback?: string,
    immediate?: boolean
  ) => void;
  onKeepSubscription: () => void;
  isProcessing?: boolean;
  className?: string;
}

// =============================================================================
// Utility Functions
// =============================================================================

export function formatPrice(amount: number, currency: string = 'USD'): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currency.toUpperCase(),
  }).format(amount / 100);
}

export function formatBillingCycle(cycle: BillingCycle): string {
  const labels: Record<BillingCycle, string> = {
    monthly: 'Monthly',
    quarterly: 'Quarterly',
    annual: 'Annual',
  };
  return labels[cycle];
}

export function formatBillingCycleShort(cycle: BillingCycle): string {
  const labels: Record<BillingCycle, string> = {
    monthly: '/mo',
    quarterly: '/quarter',
    annual: '/year',
  };
  return labels[cycle];
}

export function formatPlanTier(tier: PlanTier): string {
  const labels: Record<PlanTier, string> = {
    free: 'Free',
    starter: 'Starter',
    professional: 'Professional',
    business: 'Business',
    enterprise: 'Enterprise',
  };
  return labels[tier];
}

export function formatSubscriptionStatus(status: SubscriptionStatus): string {
  const labels: Record<SubscriptionStatus, string> = {
    trialing: 'Trial',
    active: 'Active',
    past_due: 'Past Due',
    canceled: 'Canceled',
    unpaid: 'Unpaid',
    incomplete: 'Incomplete',
    incomplete_expired: 'Expired',
    paused: 'Paused',
  };
  return labels[status];
}

export function getStatusColor(status: SubscriptionStatus): string {
  const colors: Record<SubscriptionStatus, string> = {
    trialing: 'text-blue-600 bg-blue-100',
    active: 'text-green-600 bg-green-100',
    past_due: 'text-amber-600 bg-amber-100',
    canceled: 'text-neutral-600 bg-neutral-100',
    unpaid: 'text-red-600 bg-red-100',
    incomplete: 'text-amber-600 bg-amber-100',
    incomplete_expired: 'text-red-600 bg-red-100',
    paused: 'text-neutral-600 bg-neutral-100',
  };
  return colors[status];
}

export function getPriceForCycle(plan: Plan, cycle: BillingCycle): number {
  switch (cycle) {
    case 'monthly':
      return plan.monthlyPrice;
    case 'quarterly':
      return plan.quarterlyPrice;
    case 'annual':
      return plan.annualPrice;
    default:
      return plan.monthlyPrice;
  }
}

export function calculateSavings(plan: Plan): {
  annual: number;
  annualPercent: number;
  quarterly: number;
  quarterlyPercent: number;
} {
  const monthlyAnnualized = plan.monthlyPrice * 12;
  const annualSavings = monthlyAnnualized - plan.annualPrice;
  const annualSavingsPercent =
    monthlyAnnualized > 0
      ? Math.round((annualSavings / monthlyAnnualized) * 100)
      : 0;

  const monthlyQuarterly = plan.monthlyPrice * 3;
  const quarterlySavings = monthlyQuarterly - plan.quarterlyPrice;
  const quarterlySavingsPercent =
    monthlyQuarterly > 0
      ? Math.round((quarterlySavings / monthlyQuarterly) * 100)
      : 0;

  return {
    annual: annualSavings,
    annualPercent: annualSavingsPercent,
    quarterly: quarterlySavings,
    quarterlyPercent: quarterlySavingsPercent,
  };
}

export function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

export function formatDateShort(dateString: string): string {
  return new Date(dateString).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}
