/**
 * Money helpers — the single place for currency formatting and rounding.
 *
 * Units:
 * - Invoice/quote/payment/report amounts from the API are decimal *dollars*,
 *   serialised by the Go backend (shopspring/decimal) as strings, e.g. "12.50".
 *   Use {@link formatDollars} / {@link toAmount}.
 * - Stripe/membership prices are integer *cents* (fields named `*_cents`).
 *   Use {@link formatCents}.
 * - Tax rates are stored as decimal fractions (0.07) and edited as percents (7).
 */

export const DEFAULT_CURRENCY = 'USD';
export const DEFAULT_LOCALE = 'en-US';

export interface FormatMoneyOptions {
  currency?: string;
  locale?: string;
  /** Round to whole currency units (e.g. "$1,235") for dashboards/reports. */
  wholeUnits?: boolean;
}

/** Parse an API decimal (string or number) into a finite number (else 0). */
export function toAmount(value: string | number | null | undefined): number {
  if (value === null || value === undefined || value === '') return 0;
  const num = typeof value === 'number' ? value : parseFloat(value);
  return Number.isFinite(num) ? num : 0;
}

/** Round to 2 decimal places (cents), avoiding binary float drift. */
export function roundToCents(value: number): number {
  return Math.round((value + Number.EPSILON) * 100) / 100;
}

function formatter(options: FormatMoneyOptions): Intl.NumberFormat {
  const { currency = DEFAULT_CURRENCY, locale = DEFAULT_LOCALE } = options;
  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency,
    ...(options.wholeUnits
      ? { minimumFractionDigits: 0, maximumFractionDigits: 0 }
      : { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
  });
}

/** Format a decimal-dollar amount (string or number), e.g. "12.5" → "$12.50". */
export function formatDollars(
  amount: string | number | null | undefined,
  options: FormatMoneyOptions = {}
): string {
  return formatter(options).format(toAmount(amount));
}

/** Format an integer-cents amount, e.g. 1250 → "$12.50". */
export function formatCents(
  cents: number | null | undefined,
  options: FormatMoneyOptions = {}
): string {
  return formatter(options).format(toAmount(cents) / 100);
}

/** Line total in dollars, rounded to cents. */
export function lineTotal(
  quantity: string | number,
  unitPrice: string | number
): number {
  return roundToCents(toAmount(quantity) * toAmount(unitPrice));
}

/**
 * Tax rate fraction (0.07, or "0.0700" from the API) → percent string for an
 * input ("7"), without float artefacts like "7.000000000000001".
 */
export function taxRateToPercent(
  rate: string | number | null | undefined
): string {
  const num = toAmount(rate);
  if (!num) return '0';
  return Number((num * 100).toFixed(4)).toString();
}

/** Percent input ("7" or 7) → tax rate fraction (0.07). */
export function percentToTaxRate(percent: string | number): number {
  return Number((toAmount(percent) / 100).toFixed(6));
}
