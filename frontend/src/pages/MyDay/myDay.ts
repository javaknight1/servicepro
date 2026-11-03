/**
 * Helpers for the technician's day (T173): tenant-local dates, the one big
 * status action per visit, and the maps / phone links.
 */
import type { Job, ServiceAddress } from '@services/jobService';
import { JobStatus } from '@services/jobService';

/** React Query key for the caller's visits on one day in one tenant. */
export const myDayQueryKey = (
  tenantId: string | null | undefined,
  date: string
) => ['my-day', tenantId ?? null, date] as const;

/** React Query key for one visit (the job) on the visit page. */
export const visitQueryKey = (jobId: string | null | undefined) =>
  ['visit', jobId ?? null] as const;

/** Today's date, YYYY-MM-DD, in `timeZone` (the browser's when unset). */
export function todayIn(timeZone?: string, now: Date = new Date()): string {
  // en-CA formats dates as YYYY-MM-DD.
  try {
    return new Intl.DateTimeFormat('en-CA', {
      timeZone: timeZone || undefined,
    }).format(now);
  } catch {
    return new Intl.DateTimeFormat('en-CA').format(now);
  }
}

/** `date` (YYYY-MM-DD) moved by `days` calendar days. */
export function addDays(date: string, days: number): string {
  const [y, m, d] = date.split('-').map(Number);
  const moved = new Date(Date.UTC(y, m - 1, d + days));
  return moved.toISOString().slice(0, 10);
}

/** Whether `value` is a YYYY-MM-DD date. */
export function isDate(value: string | null | undefined): value is string {
  return !!value && /^\d{4}-\d{2}-\d{2}$/.test(value);
}

/** "Monday, Nov 2" for a YYYY-MM-DD date. */
export function formatDayHeading(date: string): string {
  const [y, m, d] = date.split('-').map(Number);
  return new Intl.DateTimeFormat('en-US', {
    weekday: 'long',
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC',
  }).format(new Date(Date.UTC(y, m - 1, d)));
}

/** "5:30 PM" for an epoch-seconds instant, in the tenant's time zone. */
export function formatClock(epoch: number, timeZone?: string): string {
  const options: Intl.DateTimeFormatOptions = {
    hour: 'numeric',
    minute: '2-digit',
  };
  try {
    return new Intl.DateTimeFormat('en-US', {
      ...options,
      timeZone: timeZone || undefined,
    }).format(new Date(epoch * 1000));
  } catch {
    return new Intl.DateTimeFormat('en-US', options).format(
      new Date(epoch * 1000)
    );
  }
}

/** "in 18 min" / "in 2 hr" until `epoch`, or null once it has passed. */
export function startsIn(epoch: number, now: Date = new Date()): string | null {
  const minutes = Math.round((epoch * 1000 - now.getTime()) / 60000);
  if (minutes <= 0) return null;
  if (minutes < 60) return `in ${minutes} min`;
  const hours = Math.round(minutes / 60);
  return `in ${hours} hr`;
}

/** Statuses after the visit's work is done. */
const DONE: ReadonlySet<JobStatus> = new Set([
  JobStatus.COMPLETED,
  JobStatus.INVOICED,
  JobStatus.PAID,
]);

export function isDone(job: Pick<Job, 'status'>): boolean {
  return DONE.has(job.status);
}

/** The visit's status in the field's words (no invoicing states). */
export function visitStatusLabel(status: JobStatus): string {
  switch (status) {
    case JobStatus.NEW:
      return 'Not scheduled';
    case JobStatus.SCHEDULED:
      return 'Scheduled';
    case JobStatus.EN_ROUTE:
      return 'On the way';
    case JobStatus.IN_PROGRESS:
      return 'In progress';
    case JobStatus.ON_HOLD:
      return 'On hold';
    case JobStatus.CANCELLED:
      return 'Cancelled';
    default:
      return 'Done';
  }
}

export interface VisitAction {
  /** Button text: Start, Complete or Resume. */
  label: string;
  /** The status POST /jobs/{id}/transition moves the job to. */
  to: JobStatus;
}

/**
 * The visit's one big action (Start → Complete), from the server's state
 * machine (`allowed_transitions`): null when there is nothing to do.
 */
export function primaryAction(
  job: Pick<Job, 'status' | 'allowed_transitions'>
): VisitAction | null {
  const allowed = job.allowed_transitions ?? [];
  let action: VisitAction;
  switch (job.status) {
    case JobStatus.SCHEDULED:
    case JobStatus.EN_ROUTE:
      action = { label: 'Start', to: JobStatus.IN_PROGRESS };
      break;
    case JobStatus.IN_PROGRESS:
      action = { label: 'Complete', to: JobStatus.COMPLETED };
      break;
    case JobStatus.ON_HOLD:
      action = { label: 'Resume', to: JobStatus.IN_PROGRESS };
      break;
    default:
      return null;
  }
  return allowed.includes(action.to) ? action : null;
}

/**
 * The visit to work next: the one in progress, else the first not yet done.
 */
export function nextVisit(visits: Job[]): Job | undefined {
  return (
    visits.find((v) => v.status === JobStatus.IN_PROGRESS) ??
    visits.find((v) => !isDone(v) && v.status !== JobStatus.CANCELLED)
  );
}

/** Where the visit is: its site's address, else the job's service address. */
export function visitAddress(job: Job): ServiceAddress | undefined {
  const address = job.site?.address ?? job.service_address;
  if (!address || !address.street?.trim()) return undefined;
  return address;
}

/** "410 Elm St, Springfield, IL 62701". */
export function formatAddress(address: ServiceAddress): string {
  const cityLine = [
    address.city,
    [address.state, address.zip].filter(Boolean).join(' '),
  ]
    .map((p) => p?.trim())
    .filter(Boolean)
    .join(', ');
  return [address.street?.trim(), cityLine].filter(Boolean).join(', ');
}

/** Directions to `address` (opens the maps app on phones). */
export function mapsUrl(address: ServiceAddress): string {
  return `https://www.google.com/maps/dir/?api=1&destination=${encodeURIComponent(
    formatAddress(address)
  )}`;
}

/** A tel: link for `phone`, or null when it has no digits. */
export function telUrl(phone: string | null | undefined): string | null {
  const dialable = (phone ?? '').replace(/[^\d+]/g, '');
  return dialable.replace(/\D/g, '').length > 0 ? `tel:${dialable}` : null;
}

/** The visit's heading: its site, else the job title. */
export function visitTitle(job: Job): string {
  return job.site?.name || job.title;
}

/** The customer's name as the office knows it. */
export function customerName(job: Job): string | undefined {
  const c = job.customer;
  if (!c) return undefined;
  return (
    c.company_name?.trim() ||
    [c.first_name, c.last_name].filter(Boolean).join(' ').trim() ||
    undefined
  );
}

/** The customer's name under the visit title, unless it repeats it. */
export function customerSubtitle(job: Job): string | undefined {
  const name = customerName(job);
  return name && name !== visitTitle(job) ? name : undefined;
}

/** Who to call at the visit: the site contact, else the customer. */
export function visitContact(
  job: Job
): { name?: string; phone: string } | null {
  if (job.site?.contact_phone) {
    return { name: job.site.contact_name, phone: job.site.contact_phone };
  }
  const phone = job.customer?.phone_primary || job.customer?.phone;
  if (phone) return { name: customerName(job), phone };
  return null;
}
