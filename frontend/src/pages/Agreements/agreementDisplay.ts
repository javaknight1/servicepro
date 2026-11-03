import type { Customer } from '@services/customerService';

/** Company name, else the contact's name. */
export function customerName(
  customer:
    Pick<Customer, 'company_name' | 'first_name' | 'last_name'> | undefined
): string {
  if (!customer) return '—';
  return (
    customer.company_name ||
    `${customer.first_name} ${customer.last_name}`.trim() ||
    '—'
  );
}

/** "18:00" (the agreement's wall-clock visit time) → "6:00 PM". */
export function formatClock(hhmm: string): string {
  const [h, m] = hhmm.split(':').map(Number);
  if (!Number.isFinite(h) || !Number.isFinite(m)) return hhmm;
  const hour12 = h % 12 === 0 ? 12 : h % 12;
  return `${hour12}:${String(m).padStart(2, '0')} ${h < 12 ? 'AM' : 'PM'}`;
}

function dayKey(date: Date, timeZone?: string): string {
  return date.toLocaleDateString('en-CA', { timeZone });
}

/** "Today · 6:00 PM", "Tomorrow · 7:15 PM", "Thu, Nov 5 · 7:00 PM". */
export function formatVisit(
  iso: string,
  timeZone?: string,
  now: Date = new Date()
): string {
  const at = new Date(iso);
  const time = at.toLocaleTimeString('en-US', {
    timeZone,
    hour: 'numeric',
    minute: '2-digit',
  });
  const tomorrow = new Date(now.getTime() + 24 * 60 * 60 * 1000);
  let day: string;
  if (dayKey(at, timeZone) === dayKey(now, timeZone)) day = 'Today';
  else if (dayKey(at, timeZone) === dayKey(tomorrow, timeZone))
    day = 'Tomorrow';
  else
    day = at.toLocaleDateString('en-US', {
      timeZone,
      weekday: 'short',
      month: 'short',
      day: 'numeric',
    });
  return `${day} · ${time}`;
}
