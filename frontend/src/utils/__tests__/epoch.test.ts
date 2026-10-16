import { describe, it, expect } from 'vitest';
import {
  datetimeLocalToEpoch,
  epochToDatetimeLocal,
  epochToZonedDate,
  zonedDateToEpoch,
  formatEpochDateTime,
  formatEpochTime,
  isValidTimeZone,
  normalizeTimeZone,
} from '../epoch';

const NEW_YORK = 'America/New_York';
const HONOLULU = 'Pacific/Honolulu';

const utc = (y: number, mo: number, d: number, h: number, mi = 0): number =>
  Date.UTC(y, mo - 1, d, h, mi) / 1000;

describe('epoch timezone conversion', () => {
  describe('America/New_York (EDT, UTC-4)', () => {
    it('treats typed 09:00 as 09:00 EDT, i.e. 13:00 UTC', () => {
      expect(datetimeLocalToEpoch('2026-07-15T09:00', NEW_YORK)).toBe(
        utc(2026, 7, 15, 13)
      );
    });

    it('renders 13:00 UTC back as 09:00 in the input', () => {
      expect(epochToDatetimeLocal(utc(2026, 7, 15, 13), NEW_YORK)).toBe(
        '2026-07-15T09:00'
      );
    });

    it('uses EST (UTC-5) in winter', () => {
      expect(datetimeLocalToEpoch('2026-01-15T09:00', NEW_YORK)).toBe(
        utc(2026, 1, 15, 14)
      );
    });

    it('handles the DST transition day correctly', () => {
      // 2026-03-08: clocks jump 02:00 -> 03:00 EST->EDT
      expect(datetimeLocalToEpoch('2026-03-08T01:30', NEW_YORK)).toBe(
        utc(2026, 3, 8, 6, 30)
      );
      expect(datetimeLocalToEpoch('2026-03-08T09:00', NEW_YORK)).toBe(
        utc(2026, 3, 8, 13)
      );
    });

    it('formats display strings in the tenant timezone', () => {
      expect(formatEpochTime(utc(2026, 7, 15, 13), NEW_YORK)).toBe('9:00 AM');
      expect(formatEpochDateTime(utc(2026, 7, 15, 13), NEW_YORK)).toMatch(
        /Jul 15, 2026.*9:00\sAM/
      );
    });
  });

  describe('Pacific/Honolulu (HST, UTC-10, no DST)', () => {
    it('treats typed 09:00 as 09:00 HST, i.e. 19:00 UTC', () => {
      expect(datetimeLocalToEpoch('2026-07-15T09:00', HONOLULU)).toBe(
        utc(2026, 7, 15, 19)
      );
      expect(datetimeLocalToEpoch('2026-01-15T09:00', HONOLULU)).toBe(
        utc(2026, 1, 15, 19)
      );
    });

    it('crosses the UTC date boundary correctly', () => {
      // 20:00 HST on the 15th is 06:00 UTC on the 16th
      expect(datetimeLocalToEpoch('2026-07-15T20:00', HONOLULU)).toBe(
        utc(2026, 7, 16, 6)
      );
      expect(epochToDatetimeLocal(utc(2026, 7, 16, 6), HONOLULU)).toBe(
        '2026-07-15T20:00'
      );
    });

    it('formats display in HST', () => {
      expect(formatEpochTime(utc(2026, 7, 15, 19), HONOLULU)).toBe('9:00 AM');
    });
  });

  it('round-trips input values in any zone', () => {
    for (const tz of [NEW_YORK, HONOLULU, 'Europe/London', 'Asia/Kolkata']) {
      const value = '2026-11-02T08:45';
      const epoch = datetimeLocalToEpoch(value, tz);
      expect(epoch).not.toBeNull();
      expect(epochToDatetimeLocal(epoch, tz)).toBe(value);
    }
  });

  describe('calendar zoned dates', () => {
    it('produces Dates whose local fields are the tenant wall-clock time', () => {
      const epoch = utc(2026, 7, 15, 13); // 09:00 EDT, 03:00 HST
      const ny = epochToZonedDate(epoch, NEW_YORK);
      expect(ny.getHours()).toBe(9);
      expect(ny.getDate()).toBe(15);
      const hnl = epochToZonedDate(epoch, HONOLULU);
      expect(hnl.getHours()).toBe(3);
    });

    it('agrees with the job form for the same job', () => {
      const epoch = datetimeLocalToEpoch('2026-07-15T09:00', NEW_YORK);
      expect(epoch).not.toBeNull();
      const calendarDate = epochToZonedDate(epoch as number, NEW_YORK);
      expect(calendarDate.getHours()).toBe(9);
      expect(calendarDate.getMinutes()).toBe(0);
    });

    it('converts a zoned Date back to the original epoch', () => {
      const epoch = utc(2026, 7, 15, 19);
      expect(
        zonedDateToEpoch(epochToZonedDate(epoch, HONOLULU), HONOLULU)
      ).toBe(epoch);
    });
  });

  describe('browser-local fallback', () => {
    it('uses browser local time when no timezone is given', () => {
      const expected = new Date(2026, 6, 15, 9, 0).getTime() / 1000;
      expect(datetimeLocalToEpoch('2026-07-15T09:00')).toBe(expected);
      expect(epochToDatetimeLocal(expected)).toBe('2026-07-15T09:00');
    });

    it('falls back to browser local for an invalid timezone', () => {
      const expected = new Date(2026, 6, 15, 9, 0).getTime() / 1000;
      expect(datetimeLocalToEpoch('2026-07-15T09:00', 'Not/AZone')).toBe(
        expected
      );
      expect(isValidTimeZone('Not/AZone')).toBe(false);
      expect(normalizeTimeZone('Not/AZone')).toBeUndefined();
      expect(normalizeTimeZone(HONOLULU)).toBe(HONOLULU);
    });
  });

  it('rejects empty and malformed input', () => {
    expect(datetimeLocalToEpoch('', NEW_YORK)).toBeNull();
    expect(datetimeLocalToEpoch('not-a-date', NEW_YORK)).toBeNull();
    expect(datetimeLocalToEpoch('2026-13-01T09:00', NEW_YORK)).toBeNull();
    expect(epochToDatetimeLocal(null)).toBe('');
    expect(epochToDatetimeLocal(0)).toBe('');
  });
});
