import { describe, it, expect } from 'vitest';
import { JobStatus } from '@services/jobService';
import {
  addDays,
  formatDayHeading,
  nextVisit,
  primaryAction,
  startsIn,
  telUrl,
  todayIn,
} from '../myDay';
import type { Job } from '@services/jobService';

describe('My day helpers (T173)', () => {
  it('moves calendar dates across months and years', () => {
    expect(addDays('2026-11-30', 1)).toBe('2026-12-01');
    expect(addDays('2027-01-01', -1)).toBe('2026-12-31');
    expect(addDays('2026-03-08', 1)).toBe('2026-03-09');
  });

  it("reads today in the tenant's time zone", () => {
    // 03:00 UTC on Nov 4 is still Nov 3 in Chicago.
    const now = new Date('2026-11-04T03:00:00Z');
    expect(todayIn('America/Chicago', now)).toBe('2026-11-03');
    expect(todayIn('UTC', now)).toBe('2026-11-04');
  });

  it('formats the day heading like the mockup', () => {
    expect(formatDayHeading('2026-11-02')).toBe('Monday, Nov 2');
  });

  it('offers only what the state machine allows', () => {
    const allowed = [JobStatus.IN_PROGRESS, JobStatus.EN_ROUTE];
    expect(
      primaryAction({
        status: JobStatus.SCHEDULED,
        allowed_transitions: allowed,
      })
    ).toEqual({ label: 'Start', to: JobStatus.IN_PROGRESS });
    expect(
      primaryAction({ status: JobStatus.SCHEDULED, allowed_transitions: [] })
    ).toBeNull();
    expect(
      primaryAction({
        status: JobStatus.IN_PROGRESS,
        allowed_transitions: [JobStatus.COMPLETED],
      })
    ).toEqual({ label: 'Complete', to: JobStatus.COMPLETED });
    expect(
      primaryAction({
        status: JobStatus.ON_HOLD,
        allowed_transitions: [JobStatus.IN_PROGRESS],
      })
    ).toEqual({ label: 'Resume', to: JobStatus.IN_PROGRESS });
    expect(
      primaryAction({
        status: JobStatus.COMPLETED,
        allowed_transitions: [JobStatus.IN_PROGRESS],
      })
    ).toBeNull();
  });

  it('picks the visit in progress, else the first not done', () => {
    const v = (id: string, status: JobStatus) => ({ id, status }) as Job;
    expect(
      nextVisit([
        v('a', JobStatus.COMPLETED),
        v('b', JobStatus.SCHEDULED),
        v('c', JobStatus.IN_PROGRESS),
      ])?.id
    ).toBe('c');
    expect(
      nextVisit([v('a', JobStatus.COMPLETED), v('b', JobStatus.SCHEDULED)])?.id
    ).toBe('b');
    expect(nextVisit([v('a', JobStatus.COMPLETED)])).toBeUndefined();
  });

  it('builds dialable phone links', () => {
    expect(telUrl('(555) 010-0100')).toBe('tel:5550100100');
    expect(telUrl('+1 555 0100')).toBe('tel:+15550100');
    expect(telUrl('n/a')).toBeNull();
    expect(telUrl(undefined)).toBeNull();
  });

  it('counts down to the next visit', () => {
    const now = new Date('2026-11-03T17:12:00Z');
    const at = (iso: string) => Date.parse(iso) / 1000;
    expect(startsIn(at('2026-11-03T17:30:00Z'), now)).toBe('in 18 min');
    expect(startsIn(at('2026-11-03T19:12:00Z'), now)).toBe('in 2 hr');
    expect(startsIn(at('2026-11-03T17:00:00Z'), now)).toBeNull();
  });
});
