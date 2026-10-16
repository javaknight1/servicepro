/**
 * Epoch <-> wall-clock helpers.
 *
 * Storage is always true UTC epoch seconds. Anything a user types or reads
 * (datetime-local inputs, list/detail displays, the calendar) is wall-clock
 * time in the *display timezone*: the tenant's configured IANA timezone when
 * available, otherwise the browser's local timezone (`timeZone` undefined).
 *
 * Only `Intl` is used, so no timezone database dependency is needed.
 */

interface WallClock {
  year: number;
  month: number; // 1-12
  day: number;
  hour: number;
  minute: number;
  second: number;
}

const formatterCache = new Map<string, Intl.DateTimeFormat>();

function getPartsFormatter(timeZone: string): Intl.DateTimeFormat {
  let fmt = formatterCache.get(timeZone);
  if (!fmt) {
    fmt = new Intl.DateTimeFormat('en-US', {
      timeZone,
      hourCycle: 'h23',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
    formatterCache.set(timeZone, fmt);
  }
  return fmt;
}

/** True when `timeZone` is a valid IANA zone name understood by Intl. */
export function isValidTimeZone(timeZone: string | null | undefined): boolean {
  if (!timeZone) return false;
  try {
    getPartsFormatter(timeZone);
    return true;
  } catch {
    return false;
  }
}

/** Wall-clock fields of an instant in `timeZone` (browser local if undefined). */
function getWallClock(ms: number, timeZone?: string): WallClock {
  if (!timeZone) {
    const d = new Date(ms);
    return {
      year: d.getFullYear(),
      month: d.getMonth() + 1,
      day: d.getDate(),
      hour: d.getHours(),
      minute: d.getMinutes(),
      second: d.getSeconds(),
    };
  }
  const parts = getPartsFormatter(timeZone).formatToParts(new Date(ms));
  const get = (type: Intl.DateTimeFormatPartTypes): number =>
    Number(parts.find((p) => p.type === type)?.value ?? 0);
  return {
    year: get('year'),
    month: get('month'),
    day: get('day'),
    hour: get('hour') % 24,
    minute: get('minute'),
    second: get('second'),
  };
}

function wallClockAsUtcMs(w: WallClock): number {
  return Date.UTC(w.year, w.month - 1, w.day, w.hour, w.minute, w.second);
}

/** Offset (ms) of `timeZone` from UTC at instant `ms`. */
function getOffsetMs(ms: number, timeZone: string): number {
  const wholeSecondMs = Math.floor(ms / 1000) * 1000;
  return (
    wallClockAsUtcMs(getWallClock(wholeSecondMs, timeZone)) - wholeSecondMs
  );
}

/** Convert wall-clock fields in `timeZone` (browser local if undefined) to epoch ms. */
function wallClockToMs(w: WallClock, timeZone?: string): number {
  if (!timeZone) {
    return new Date(
      w.year,
      w.month - 1,
      w.day,
      w.hour,
      w.minute,
      w.second
    ).getTime();
  }
  const asUtc = wallClockAsUtcMs(w);
  // Two passes handle DST transitions around the target instant.
  let ms = asUtc - getOffsetMs(asUtc, timeZone);
  const secondOffset = getOffsetMs(ms, timeZone);
  ms = asUtc - secondOffset;
  return ms;
}

function pad(n: number): string {
  return String(n).padStart(2, '0');
}

/**
 * Parse a datetime-local input value (YYYY-MM-DDThh:mm) as wall-clock time in
 * `timeZone` (browser local when undefined) and return UTC epoch seconds.
 * Returns null if the value is empty or invalid.
 */
export function datetimeLocalToEpoch(
  value: string,
  timeZone?: string
): number | null {
  if (!value) return null;
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/.exec(
    value
  );
  if (!match) return null;
  const [, y, mo, d, h, mi, s] = match;
  const wall: WallClock = {
    year: Number(y),
    month: Number(mo),
    day: Number(d),
    hour: Number(h),
    minute: Number(mi),
    second: s ? Number(s) : 0,
  };
  if (
    wall.month < 1 ||
    wall.month > 12 ||
    wall.day < 1 ||
    wall.day > 31 ||
    wall.hour > 23 ||
    wall.minute > 59
  ) {
    return null;
  }
  const ms = wallClockToMs(wall, normalizeTimeZone(timeZone));
  if (isNaN(ms)) return null;
  return Math.floor(ms / 1000);
}

/**
 * Convert epoch seconds to a datetime-local input value (YYYY-MM-DDThh:mm)
 * in `timeZone` (browser local when undefined).
 * Returns empty string if epoch is null/undefined/0.
 */
export function epochToDatetimeLocal(
  epoch: number | null | undefined,
  timeZone?: string
): string {
  if (!epoch) return '';
  const w = getWallClock(epoch * 1000, normalizeTimeZone(timeZone));
  return `${w.year}-${pad(w.month)}-${pad(w.day)}T${pad(w.hour)}:${pad(w.minute)}`;
}

/**
 * Format epoch seconds as a human-readable date+time in `timeZone`
 * (browser local when undefined). e.g. "Jan 15, 2024, 9:00 AM"
 */
export function formatEpochDateTime(epoch: number, timeZone?: string): string {
  return new Date(epoch * 1000).toLocaleString('en-US', {
    timeZone: normalizeTimeZone(timeZone),
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  });
}

/**
 * Format epoch seconds as a human-readable time in `timeZone`
 * (browser local when undefined). e.g. "9:00 AM"
 */
export function formatEpochTime(epoch: number, timeZone?: string): string {
  return new Date(epoch * 1000).toLocaleTimeString('en-US', {
    timeZone: normalizeTimeZone(timeZone),
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  });
}

/**
 * Convert epoch seconds to a Date whose *browser-local* fields equal the
 * wall-clock time in `timeZone`. Use this for components (e.g.
 * react-big-calendar) that always render Dates in browser-local time, so they
 * display tenant-timezone times. Inverse: {@link zonedDateToEpoch}.
 */
export function epochToZonedDate(epoch: number, timeZone?: string): Date {
  const tz = normalizeTimeZone(timeZone);
  if (!tz) return new Date(epoch * 1000);
  const w = getWallClock(epoch * 1000, tz);
  return new Date(w.year, w.month - 1, w.day, w.hour, w.minute, w.second);
}

/** Inverse of {@link epochToZonedDate}: read local fields as `timeZone` wall time. */
export function zonedDateToEpoch(date: Date, timeZone?: string): number {
  const tz = normalizeTimeZone(timeZone);
  if (!tz) return dateToEpoch(date);
  return Math.floor(
    wallClockToMs(
      {
        year: date.getFullYear(),
        month: date.getMonth() + 1,
        day: date.getDate(),
        hour: date.getHours(),
        minute: date.getMinutes(),
        second: date.getSeconds(),
      },
      tz
    ) / 1000
  );
}

/** Returns the zone if valid, otherwise undefined (browser local). */
export function normalizeTimeZone(
  timeZone: string | null | undefined
): string | undefined {
  return timeZone && isValidTimeZone(timeZone) ? timeZone : undefined;
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
