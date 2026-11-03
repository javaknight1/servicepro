import type { AgingBucket } from '@app-types/arAging';

/** A/R aging buckets, youngest first. */
export const AGING_BUCKETS: AgingBucket[] = [
  'current',
  '1-30',
  '31-60',
  '61-90',
  '90+',
];

/**
 * Colours from the MVP mockups (05, 11): green while current, warming through
 * orange to red as the debt ages. `bar` fills the bar and dots, `border` is the
 * bucket column's left rule, `badge` the per-customer bucket pill.
 */
export const AGING_BUCKET_STYLES: Record<
  AgingBucket,
  {
    label: string;
    shortLabel: string;
    bar: string;
    border: string;
    badge: string;
  }
> = {
  current: {
    label: 'Current',
    shortLabel: 'Current',
    bar: 'bg-success-600',
    border: 'border-success-600',
    badge: 'bg-success-50 text-success-700 border-success-200',
  },
  '1-30': {
    label: '1–30 days',
    shortLabel: '1–30 days',
    bar: 'bg-primary-300',
    border: 'border-primary-300',
    badge: 'bg-primary-50 text-primary-700 border-primary-200',
  },
  '31-60': {
    label: '31–60 days',
    shortLabel: '31–60',
    bar: 'bg-primary-500',
    border: 'border-primary-500',
    badge: 'bg-warning-50 text-warning-800 border-warning-200',
  },
  '61-90': {
    label: '61–90 days',
    shortLabel: '61–90',
    bar: 'bg-primary-700',
    border: 'border-primary-700',
    badge: 'bg-warning-100 text-warning-800 border-warning-300',
  },
  '90+': {
    label: '90+ days',
    shortLabel: '90+',
    bar: 'bg-error-600',
    border: 'border-error-600',
    badge: 'bg-error-50 text-error-700 border-error-200',
  },
};
