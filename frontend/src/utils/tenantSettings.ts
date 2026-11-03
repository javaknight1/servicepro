/**
 * Organization settings helpers (T235): the choices the settings screens and
 * the onboarding wizard offer, and client-side checks that mirror the API's
 * validation (the API stays the authority and answers 400 with field errors).
 */
import type { Tenant, TenantSettings } from '@/types/tenant';

/** Longest invoice/quote/job number prefix the API accepts. */
export const MAX_NUMBER_PREFIX_LENGTH = 20;

/** Default number prefixes the API uses when none is set. */
export const DEFAULT_PREFIXES = {
  invoicePrefix: 'INV-',
  quotePrefix: 'QUO-',
  jobPrefix: 'JOB-',
} as const;

/** Active ISO 4217 codes the API accepts (backend models/currencies.go). */
const CURRENCY_CODES: readonly string[] = [
  'AED AFN ALL AMD ANG AOA ARS AUD AWG AZN BAM BBD BDT BGN BHD BIF BMD',
  'BND BOB BRL BSD BTN BWP BYN BZD CAD CDF CHF CLP CNY COP CRC CUP CVE',
  'CZK DJF DKK DOP DZD EGP ERN ETB EUR FJD FKP GBP GEL GHS GIP GMD GNF',
  'GTQ GYD HKD HNL HTG HUF IDR ILS INR IQD IRR ISK JMD JOD JPY KES KGS',
  'KHR KMF KPW KRW KWD KYD KZT LAK LBP LKR LRD LSL LYD MAD MDL MGA MKD',
  'MMK MNT MOP MRU MUR MVR MWK MXN MYR MZN NAD NGN NIO NOK NPR NZD OMR',
  'PAB PEN PGK PHP PKR PLN PYG QAR RON RSD RUB RWF SAR SBD SCR SDG SEK',
  'SGD SHP SLE SOS SRD SSP STN SVC SYP SZL THB TJS TMT TND TOP TRY TTD',
  'TWD TZS UAH UGX USD UYU UZS VES VND VUV WST XAF XCD XOF XPF YER ZAR',
  'ZMW ZWG',
]
  .join(' ')
  .split(' ');

/** Currencies listed first: the US/Canada buyer (D7) and the common ones. */
const COMMON_CURRENCIES = ['USD', 'CAD', 'EUR', 'GBP', 'AUD'];

function currencyName(code: string): string {
  try {
    return new Intl.DisplayNames(['en'], { type: 'currency' }).of(code) ?? code;
  } catch {
    return code;
  }
}

export interface Option {
  value: string;
  label: string;
}

/** Currency select options: common ones first, then the rest A–Z. */
export const CURRENCY_OPTIONS: Option[] = [
  ...COMMON_CURRENCIES,
  ...CURRENCY_CODES.filter((c) => !COMMON_CURRENCIES.includes(c)),
].map((code) => ({ value: code, label: `${code} · ${currencyName(code)}` }));

/** Time zones offered in the pickers (IANA names). */
const TIMEZONES: Option[] = [
  { value: 'America/New_York', label: 'Eastern Time (America/New_York)' },
  { value: 'America/Chicago', label: 'Central Time (America/Chicago)' },
  { value: 'America/Denver', label: 'Mountain Time (America/Denver)' },
  { value: 'America/Phoenix', label: 'Arizona (America/Phoenix)' },
  { value: 'America/Los_Angeles', label: 'Pacific Time (America/Los_Angeles)' },
  { value: 'America/Anchorage', label: 'Alaska (America/Anchorage)' },
  { value: 'Pacific/Honolulu', label: 'Hawaii (Pacific/Honolulu)' },
  { value: 'America/Halifax', label: 'Atlantic Time (America/Halifax)' },
  { value: 'America/Toronto', label: 'Toronto (America/Toronto)' },
  { value: 'America/Vancouver', label: 'Vancouver (America/Vancouver)' },
  { value: 'Europe/London', label: 'London (Europe/London)' },
  { value: 'UTC', label: 'UTC' },
];

/** Time zone options, including `current` when it is not in the list. */
export function timezoneOptions(current?: string): Option[] {
  if (!current || TIMEZONES.some((z) => z.value === current)) return TIMEZONES;
  return [...TIMEZONES, { value: current, label: current }];
}

const CONTROL_CHARS = /\p{Cc}/u;

/** Problem with a number prefix, or undefined when it is fine. */
export function prefixError(value: string): string | undefined {
  if ([...value.trim()].length > MAX_NUMBER_PREFIX_LENGTH) {
    return `Use at most ${MAX_NUMBER_PREFIX_LENGTH} characters`;
  }
  if (CONTROL_CHARS.test(value)) return 'Remove the special characters';
  return undefined;
}

/**
 * The full settings to save: the API replaces settings as a whole, so a
 * partial change is merged onto everything the tenant already has.
 */
export function mergeSettings(
  tenant: Pick<Tenant, 'settings'>,
  patch: Partial<TenantSettings>
): TenantSettings {
  return { ...(tenant.settings ?? {}), ...patch };
}
/** The zone the browser runs in, when it is one we offer. */
export function browserTimezone(): string {
  try {
    const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    if (TIMEZONES.some((z) => z.value === zone)) return zone;
  } catch {
    // fall through
  }
  return 'America/New_York';
}

/** Trades offered in onboarding step 1 (D7). */
export const TRADE_OPTIONS = [
  {
    value: 'commercial_cleaning',
    label: 'Commercial cleaning',
    description: 'Janitorial, day porter, floor care',
  },
  {
    value: 'facility_maintenance',
    label: 'Facility maintenance',
    description: 'Preventive maintenance, repairs, work orders',
  },
  {
    value: 'other',
    label: 'Something else',
    description: 'Another kind of commercial service work',
  },
] as const;
