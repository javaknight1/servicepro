import { StripeError } from '@stripe/stripe-js';

// =============================================================================
// Payment Types
// =============================================================================

export type PaymentStatus =
  | 'idle'
  | 'processing'
  | 'succeeded'
  | 'failed'
  | 'requires_action'
  | 'canceled';

export type PaymentMethod = 'card' | 'bank_transfer' | 'invoice';

export interface PaymentAmount {
  amount: number;
  currency: string;
  displayAmount: string;
}

export interface LineItem {
  id: string;
  description: string;
  quantity: number;
  unitPrice: number;
  totalPrice: number;
}

export interface PaymentSummaryData {
  lineItems: LineItem[];
  subtotal: number;
  tax: number;
  taxRate?: number;
  discount?: number;
  discountCode?: string;
  total: number;
  currency: string;
}

// =============================================================================
// Billing Address Types
// =============================================================================

export interface BillingAddressData {
  name: string;
  email: string;
  phone?: string;
  line1: string;
  line2?: string;
  city: string;
  state: string;
  postalCode: string;
  country: string;
}

export interface BillingAddressErrors {
  name?: string;
  email?: string;
  phone?: string;
  line1?: string;
  line2?: string;
  city?: string;
  state?: string;
  postalCode?: string;
  country?: string;
}

// =============================================================================
// Payment Result Types
// =============================================================================

export interface PaymentResultData {
  status: PaymentStatus;
  paymentIntentId?: string;
  amount?: PaymentAmount;
  error?: StripeError | Error | string;
  receiptUrl?: string;
  receiptEmail?: string;
  last4?: string;
  brand?: string;
}

// =============================================================================
// Payment Form Types
// =============================================================================

export interface PaymentFormData {
  billingAddress: BillingAddressData;
  savePaymentMethod?: boolean;
}

export interface PaymentFormProps {
  /** Amount in cents */
  amount: number;
  /** Currency code (e.g., 'usd') */
  currency: string;
  /** Payment summary data for display */
  summary?: PaymentSummaryData;
  /** Stripe client secret for payment intent */
  clientSecret: string;
  /** Callback when payment succeeds */
  onSuccess?: (result: PaymentResultData) => void;
  /** Callback when payment fails */
  onError?: (error: StripeError | Error) => void;
  /** Callback when payment is cancelled */
  onCancel?: () => void;
  /** Whether to show billing address form */
  collectBillingAddress?: boolean;
  /** Pre-filled billing address */
  defaultBillingAddress?: Partial<BillingAddressData>;
  /** Custom submit button text */
  submitButtonText?: string;
  /** Show save payment method checkbox */
  allowSavePaymentMethod?: boolean;
  /** Additional CSS classes */
  className?: string;
}

// =============================================================================
// Credit Card Input Types
// =============================================================================

export interface CreditCardInputProps {
  /** Whether the input is disabled */
  disabled?: boolean;
  /** Error message to display */
  error?: string;
  /** Callback when card element changes */
  onChange?: (event: {
    complete: boolean;
    empty: boolean;
    error?: StripeError;
  }) => void;
  /** Callback when card element is ready */
  onReady?: () => void;
  /** Callback when card element receives focus */
  onFocus?: () => void;
  /** Callback when card element loses focus */
  onBlur?: () => void;
  /** Additional CSS classes */
  className?: string;
}

// =============================================================================
// Billing Address Component Types
// =============================================================================

export interface BillingAddressProps {
  /** Current billing address values */
  value: BillingAddressData;
  /** Validation errors */
  errors?: BillingAddressErrors;
  /** Callback when values change */
  onChange: (address: BillingAddressData) => void;
  /** Whether the form is disabled */
  disabled?: boolean;
  /** Additional CSS classes */
  className?: string;
}

// =============================================================================
// Payment Summary Component Types
// =============================================================================

export interface PaymentSummaryProps {
  /** Payment summary data */
  summary: PaymentSummaryData;
  /** Whether to show line items */
  showLineItems?: boolean;
  /** Whether component is in loading state */
  isLoading?: boolean;
  /** Additional CSS classes */
  className?: string;
}

// =============================================================================
// Payment Result Component Types
// =============================================================================

export interface PaymentResultProps {
  /** Payment result data */
  result: PaymentResultData;
  /** Callback when user wants to try again */
  onRetry?: () => void;
  /** Callback when user wants to close/continue */
  onClose?: () => void;
  /** Custom success message */
  successMessage?: string;
  /** Custom failure message */
  failureMessage?: string;
  /** Additional CSS classes */
  className?: string;
}

// =============================================================================
// Country Options
// =============================================================================

export interface CountryOption {
  code: string;
  name: string;
}

export const COUNTRIES: CountryOption[] = [
  { code: 'US', name: 'United States' },
  { code: 'CA', name: 'Canada' },
  { code: 'GB', name: 'United Kingdom' },
  { code: 'AU', name: 'Australia' },
  { code: 'DE', name: 'Germany' },
  { code: 'FR', name: 'France' },
  { code: 'ES', name: 'Spain' },
  { code: 'IT', name: 'Italy' },
  { code: 'NL', name: 'Netherlands' },
  { code: 'BE', name: 'Belgium' },
  { code: 'AT', name: 'Austria' },
  { code: 'CH', name: 'Switzerland' },
  { code: 'IE', name: 'Ireland' },
  { code: 'NZ', name: 'New Zealand' },
  { code: 'SE', name: 'Sweden' },
  { code: 'NO', name: 'Norway' },
  { code: 'DK', name: 'Denmark' },
  { code: 'FI', name: 'Finland' },
  { code: 'PT', name: 'Portugal' },
  { code: 'JP', name: 'Japan' },
  { code: 'SG', name: 'Singapore' },
  { code: 'HK', name: 'Hong Kong' },
];

// US States
export interface StateOption {
  code: string;
  name: string;
}

export const US_STATES: StateOption[] = [
  { code: 'AL', name: 'Alabama' },
  { code: 'AK', name: 'Alaska' },
  { code: 'AZ', name: 'Arizona' },
  { code: 'AR', name: 'Arkansas' },
  { code: 'CA', name: 'California' },
  { code: 'CO', name: 'Colorado' },
  { code: 'CT', name: 'Connecticut' },
  { code: 'DE', name: 'Delaware' },
  { code: 'FL', name: 'Florida' },
  { code: 'GA', name: 'Georgia' },
  { code: 'HI', name: 'Hawaii' },
  { code: 'ID', name: 'Idaho' },
  { code: 'IL', name: 'Illinois' },
  { code: 'IN', name: 'Indiana' },
  { code: 'IA', name: 'Iowa' },
  { code: 'KS', name: 'Kansas' },
  { code: 'KY', name: 'Kentucky' },
  { code: 'LA', name: 'Louisiana' },
  { code: 'ME', name: 'Maine' },
  { code: 'MD', name: 'Maryland' },
  { code: 'MA', name: 'Massachusetts' },
  { code: 'MI', name: 'Michigan' },
  { code: 'MN', name: 'Minnesota' },
  { code: 'MS', name: 'Mississippi' },
  { code: 'MO', name: 'Missouri' },
  { code: 'MT', name: 'Montana' },
  { code: 'NE', name: 'Nebraska' },
  { code: 'NV', name: 'Nevada' },
  { code: 'NH', name: 'New Hampshire' },
  { code: 'NJ', name: 'New Jersey' },
  { code: 'NM', name: 'New Mexico' },
  { code: 'NY', name: 'New York' },
  { code: 'NC', name: 'North Carolina' },
  { code: 'ND', name: 'North Dakota' },
  { code: 'OH', name: 'Ohio' },
  { code: 'OK', name: 'Oklahoma' },
  { code: 'OR', name: 'Oregon' },
  { code: 'PA', name: 'Pennsylvania' },
  { code: 'RI', name: 'Rhode Island' },
  { code: 'SC', name: 'South Carolina' },
  { code: 'SD', name: 'South Dakota' },
  { code: 'TN', name: 'Tennessee' },
  { code: 'TX', name: 'Texas' },
  { code: 'UT', name: 'Utah' },
  { code: 'VT', name: 'Vermont' },
  { code: 'VA', name: 'Virginia' },
  { code: 'WA', name: 'Washington' },
  { code: 'WV', name: 'West Virginia' },
  { code: 'WI', name: 'Wisconsin' },
  { code: 'WY', name: 'Wyoming' },
  { code: 'DC', name: 'District of Columbia' },
];

// =============================================================================
// Utility Functions
// =============================================================================

export function formatCurrency(amount: number, currency: string): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currency.toUpperCase(),
  }).format(amount / 100);
}

export function formatCardBrand(brand: string): string {
  const brands: Record<string, string> = {
    visa: 'Visa',
    mastercard: 'Mastercard',
    amex: 'American Express',
    discover: 'Discover',
    diners: 'Diners Club',
    jcb: 'JCB',
    unionpay: 'UnionPay',
  };
  return brands[brand.toLowerCase()] || brand;
}

export const DEFAULT_BILLING_ADDRESS: BillingAddressData = {
  name: '',
  email: '',
  phone: '',
  line1: '',
  line2: '',
  city: '',
  state: '',
  postalCode: '',
  country: 'US',
};

// =============================================================================
// Stored Payment Method Types
// =============================================================================

export type PaymentMethodType = 'card' | 'bank_account' | 'us_bank_account';

export type CardBrand =
  | 'visa'
  | 'mastercard'
  | 'amex'
  | 'discover'
  | 'diners'
  | 'jcb'
  | 'unionpay'
  | 'unknown';

export interface StoredPaymentMethod {
  id: string;
  type: PaymentMethodType;
  isDefault: boolean;
  displayName: string;
  // Card details
  cardBrand?: CardBrand;
  cardLast4?: string;
  cardExpMonth?: number;
  cardExpYear?: number;
  isExpired: boolean;
  isExpiringSoon: boolean;
  // Bank details
  bankName?: string;
  bankLast4?: string;
  // Billing
  billingName?: string;
  billingEmail?: string;
  // Metadata
  nickname?: string;
  // Status
  isActive: boolean;
  lastUsedAt?: string;
  createdAt: string;
}

export interface PaymentMethodCardProps {
  /** The stored payment method data */
  paymentMethod: StoredPaymentMethod;
  /** Whether this card is selected */
  isSelected?: boolean;
  /** Callback when card is clicked/selected */
  onSelect?: (paymentMethod: StoredPaymentMethod) => void;
  /** Callback to edit the payment method */
  onEdit?: (paymentMethod: StoredPaymentMethod) => void;
  /** Callback to delete the payment method */
  onDelete?: (paymentMethod: StoredPaymentMethod) => void;
  /** Callback to set as default */
  onSetDefault?: (paymentMethod: StoredPaymentMethod) => void;
  /** Whether actions are disabled */
  disabled?: boolean;
  /** Additional CSS classes */
  className?: string;
}

export interface DefaultPaymentToggleProps {
  /** Whether the payment method is currently default */
  isDefault: boolean;
  /** Callback when toggle is clicked */
  onChange: (isDefault: boolean) => void;
  /** Whether the toggle is disabled */
  disabled?: boolean;
  /** Size of the toggle */
  size?: 'sm' | 'md' | 'lg';
  /** Additional CSS classes */
  className?: string;
}

export interface AddPaymentMethodProps {
  /** Stripe publishable key */
  stripePublishableKey: string;
  /** Customer ID for attaching payment method */
  customerId: string;
  /** Callback when payment method is added successfully */
  onSuccess?: (paymentMethod: StoredPaymentMethod) => void;
  /** Callback when adding fails */
  onError?: (error: Error) => void;
  /** Callback when modal is closed/cancelled */
  onCancel?: () => void;
  /** Whether the modal is open */
  isOpen: boolean;
  /** Whether to allow setting as default */
  allowSetDefault?: boolean;
  /** Additional CSS classes */
  className?: string;
}

export interface PaymentMethodListProps {
  /** List of stored payment methods */
  paymentMethods: StoredPaymentMethod[];
  /** Currently selected payment method ID */
  selectedId?: string;
  /** Callback when a payment method is selected */
  onSelect?: (paymentMethod: StoredPaymentMethod) => void;
  /** Callback to add a new payment method */
  onAdd?: () => void;
  /** Callback to edit a payment method */
  onEdit?: (paymentMethod: StoredPaymentMethod) => void;
  /** Callback to delete a payment method */
  onDelete?: (paymentMethod: StoredPaymentMethod) => void;
  /** Callback to set a payment method as default */
  onSetDefault?: (paymentMethod: StoredPaymentMethod) => void;
  /** Whether the list is loading */
  isLoading?: boolean;
  /** Error message to display */
  error?: string;
  /** Empty state message */
  emptyMessage?: string;
  /** Whether to show add button */
  showAddButton?: boolean;
  /** Additional CSS classes */
  className?: string;
}

// =============================================================================
// Payment Method API Types
// =============================================================================

export interface CreatePaymentMethodRequest {
  stripePaymentMethodId: string;
  nickname?: string;
  setAsDefault?: boolean;
}

export interface UpdatePaymentMethodRequest {
  nickname?: string;
  setAsDefault?: boolean;
}

export interface PaymentMethodActivityLog {
  id: string;
  paymentMethodId: string;
  customerId: string;
  activityType: string;
  description: string;
  metadata?: Record<string, unknown>;
  ipAddress?: string;
  userAgent?: string;
  createdAt: string;
}

// =============================================================================
// Tenant Billing Types (for Subscription Payment Methods)
// =============================================================================

export interface SavedPaymentMethod {
  id: string;
  card_brand: string;
  card_last_four: string;
  card_exp_month: number;
  card_exp_year: number;
  is_default: boolean;
  created_at: string;
}

export interface SavedPaymentMethodsResponse {
  payment_methods: SavedPaymentMethod[];
}

export interface AddPaymentMethodRequest {
  payment_method_id: string;
  set_as_default?: boolean;
}

export interface SetupIntentResponse {
  client_secret: string;
}

export type BillingEventStatus = 'succeeded' | 'failed' | 'pending';

export interface BillingEvent {
  id: string;
  event_type: string;
  amount_cents?: number;
  currency: string;
  status: BillingEventStatus;
  description?: string;
  stripe_invoice_url?: string;
  created_at: string;
}

export interface BillingEventsResponse {
  events: BillingEvent[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface BillingConfig {
  stripe_enabled: boolean;
  publishable_key?: string;
}
