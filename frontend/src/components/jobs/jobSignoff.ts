/** Helpers for the customer sign-off UI (T176). */
import { isAxiosError } from 'axios';
import { useQuery } from '@tanstack/react-query';
import {
  jobSignoffQueryKey,
  jobSignoffService,
} from '@services/jobSignoffService';

/** "6:58 PM", or "Mon, Nov 2 at 6:58 PM" when not today. */
export function signoffTime(
  iso: string,
  timeZone?: string,
  now: Date = new Date()
): string {
  const d = new Date(iso);
  const day = (x: Date) =>
    x.toLocaleDateString('en-US', {
      timeZone,
      year: 'numeric',
      month: 'numeric',
      day: 'numeric',
    });
  const time = d.toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    timeZone,
  });
  if (day(d) === day(now)) return time;
  const date = d.toLocaleDateString('en-US', {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    timeZone,
  });
  return `${date} at ${time}`;
}

/** The job's sign-off; disabled (no request) without a job. A 404 from a
 * server without object storage means the feature isn't available. */
export function useJobSignoff(jobId: string | undefined) {
  return useQuery({
    queryKey: jobSignoffQueryKey(jobId ?? ''),
    queryFn: () => jobSignoffService.get(jobId as string),
    enabled: !!jobId,
    // The signature link is presigned for an hour.
    staleTime: 10 * 60 * 1000,
    refetchInterval: 30 * 60 * 1000,
    retry: (count, err) =>
      !(isAxiosError(err) && err.response?.status === 404) && count < 2,
  });
}

/** Opens the service report: a new tab opened synchronously (so the popup
 * isn't blocked) and pointed at the presigned link once it arrives. */
export async function openServiceReport(jobId: string): Promise<void> {
  const tab = window.open('', '_blank');
  try {
    const url = await jobSignoffService.reportLink(jobId);
    if (tab) {
      tab.opener = null;
      tab.location.href = url;
    } else {
      window.location.assign(url);
    }
  } catch (err: unknown) {
    tab?.close();
    throw err;
  }
}
