/**
 * Parse a datetime-local input value (YYYY-MM-DDThh:mm) as UTC and return epoch seconds.
 * Returns null if the value is empty or invalid.
 */
export function datetimeLocalToEpoch(value: string): number | null {
  if (!value) return null;
  // datetime-local gives "YYYY-MM-DDThh:mm" — treat as UTC
  const date = new Date(value + ':00Z');
  if (isNaN(date.getTime())) return null;
  return Math.floor(date.getTime() / 1000);
}

/**
 * Convert epoch seconds to a datetime-local input value (YYYY-MM-DDThh:mm) in UTC.
 * Returns empty string if epoch is null/undefined/0.
 */
export function epochToDatetimeLocal(epoch: number | null | undefined): string {
  if (!epoch) return '';
  const date = new Date(epoch * 1000);
  const y = date.getUTCFullYear();
  const m = String(date.getUTCMonth() + 1).padStart(2, '0');
  const d = String(date.getUTCDate()).padStart(2, '0');
  const h = String(date.getUTCHours()).padStart(2, '0');
  const min = String(date.getUTCMinutes()).padStart(2, '0');
  return `${y}-${m}-${d}T${h}:${min}`;
}

/**
 * Format epoch seconds as a human-readable UTC date+time string.
 * e.g. "Jan 15, 2024 9:00 AM"
 */
export function formatEpochDateTimeUTC(epoch: number): string {
  const date = new Date(epoch * 1000);
  return date.toLocaleDateString('en-US', {
    timeZone: 'UTC',
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  });
}

/**
 * Format epoch seconds as a human-readable UTC time string.
 * e.g. "9:00 AM"
 */
export function formatEpochTimeUTC(epoch: number): string {
  const date = new Date(epoch * 1000);
  return date.toLocaleTimeString('en-US', {
    timeZone: 'UTC',
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  });
}

/**
 * Convert a Date object to epoch seconds.
 */
export function dateToEpoch(date: Date): number {
  return Math.floor(date.getTime() / 1000);
}

/**
 * Convert epoch seconds to a Date object.
 */
export function epochToDate(epoch: number): Date {
  return new Date(epoch * 1000);
}
