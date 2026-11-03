/**
 * The technician's day (T173, mockup 15): today's visits in time order, each
 * with its site, how to get in, who to call and one big action (Start →
 * Complete). Phone-first; never shows money.
 */
import { Link, useSearchParams } from 'react-router';
import { useQuery } from '@tanstack/react-query';
import {
  CalendarCheck,
  CheckCircle2,
  Clock,
  MapPin,
  Navigation,
  Phone,
  Play,
} from 'lucide-react';
import { DashboardLayout } from '@components/layout';
import { useTenantStore } from '@store';
import { jobService, JobStatus, type Job } from '@services/jobService';
import { cn } from '@utils/cn';
import { getErrorMessage } from '@/utils/error';
import {
  addDays,
  customerSubtitle,
  formatAddress,
  formatClock,
  formatDayHeading,
  isDate,
  isDone,
  mapsUrl,
  myDayQueryKey,
  nextVisit,
  primaryAction,
  startsIn,
  telUrl,
  todayIn,
  visitAddress,
  visitContact,
  visitStatusLabel,
  visitTitle,
} from './myDay';
import { useVisitAction } from './useVisitAction';

const DAYS = [
  { label: 'Yesterday', offset: -1 },
  { label: 'Today', offset: 0 },
  { label: 'Tomorrow', offset: 1 },
] as const;

/** Status pill colours (field wording, no invoicing states). */
function statusTone(status: JobStatus): string {
  switch (status) {
    case JobStatus.IN_PROGRESS:
      return 'border-warning-200 bg-warning-50 text-warning-700';
    case JobStatus.ON_HOLD:
      return 'border-neutral-300 bg-neutral-100 text-neutral-700';
    case JobStatus.EN_ROUTE:
      return 'border-primary-200 bg-primary-50 text-primary-700';
    case JobStatus.SCHEDULED:
    case JobStatus.NEW:
      return 'border-neutral-200 bg-white text-neutral-600';
    default:
      return 'border-success-200 bg-success-50 text-success-700';
  }
}

function StatusPill({ status }: { status: JobStatus }) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full border px-2.5 py-0.5 text-sm font-medium',
        statusTone(status)
      )}
    >
      {visitStatusLabel(status)}
    </span>
  );
}

interface VisitCardProps {
  job: Job;
  isNext: boolean;
  isToday: boolean;
  timeZone?: string;
  date: string;
  action: ReturnType<typeof useVisitAction>;
}

function VisitCard({
  job,
  isNext,
  isToday,
  timeZone,
  date,
  action,
}: VisitCardProps) {
  const done = isDone(job);
  const next = primaryAction(job);
  const address = visitAddress(job);
  const contact = visitContact(job);
  const callUrl = contact ? telUrl(contact.phone) : null;
  const customer = customerSubtitle(job);
  const start = job.scheduled_start_at;
  const soon =
    isNext && isToday && start && job.status !== JobStatus.IN_PROGRESS
      ? startsIn(start)
      : null;
  const busy = action.pendingJobId === job.id;
  const failed = action.failedJobId === job.id;
  const detailPath = `/my-day/${job.id}?date=${date}`;

  return (
    <li
      className={cn(
        'rounded-2xl border bg-white',
        isNext
          ? 'border-2 border-primary-600 p-4 shadow-sm'
          : 'border-neutral-200 p-4',
        done && 'opacity-75'
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          {isNext ? (
            <span className="mb-2 inline-flex items-center rounded-full border border-primary-200 bg-primary-50 px-2.5 py-0.5 text-sm font-medium text-primary-700">
              {job.status === JobStatus.IN_PROGRESS
                ? 'In progress'
                : soon
                  ? `Next · ${soon}`
                  : 'Next'}
            </span>
          ) : (
            <div className="mb-2">
              <StatusPill status={job.status} />
            </div>
          )}
          <Link
            to={detailPath}
            className="block text-xl font-bold leading-tight text-neutral-900 hover:underline"
          >
            {visitTitle(job)}
          </Link>
          {customer && <p className="mt-0.5 text-neutral-600">{customer}</p>}
          {job.site && job.title && (
            <p className="mt-0.5 text-sm text-neutral-500">{job.title}</p>
          )}
        </div>
        <div className="shrink-0 text-right">
          {start ? (
            <p className="font-semibold text-neutral-900">
              {formatClock(start, timeZone)}
            </p>
          ) : (
            <p className="text-sm text-neutral-500">No time set</p>
          )}
          {done && job.actual_end_at && (
            <p className="mt-1 flex items-center justify-end gap-1 text-sm text-success-700">
              <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
              Done {formatClock(job.actual_end_at, timeZone)}
            </p>
          )}
        </div>
      </div>

      {address && (
        <a
          href={mapsUrl(address)}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-3 flex min-h-11 items-center gap-2 text-neutral-700 hover:text-primary-700"
        >
          <MapPin className="h-5 w-5 shrink-0 text-neutral-400" />
          <span className="underline decoration-neutral-300 underline-offset-2">
            {formatAddress(address)}
          </span>
        </a>
      )}

      {!done && job.site?.access_notes && (
        <p className="mt-2 rounded-xl border border-warning-200 bg-warning-50 px-3 py-2 text-[15px] text-warning-800">
          Site note: {job.site.access_notes}
        </p>
      )}

      {!done && contact && callUrl && (
        <a
          href={callUrl}
          className="mt-2 flex min-h-11 items-center gap-2 text-neutral-700 hover:text-primary-700"
        >
          <Phone className="h-5 w-5 shrink-0 text-neutral-400" />
          <span>
            {contact.name ? `${contact.name} · ` : ''}
            <span className="underline decoration-neutral-300 underline-offset-2">
              {contact.phone}
            </span>
          </span>
        </a>
      )}

      {next && (
        <div className="mt-4 flex gap-3">
          <button
            type="button"
            onClick={() => action.run(job, next)}
            disabled={busy}
            aria-label={`${next.label} ${visitTitle(job)}`}
            className={cn(
              'flex min-h-14 flex-1 items-center justify-center gap-2 rounded-xl text-lg font-semibold transition-colors disabled:opacity-60',
              isNext
                ? 'bg-primary-600 text-white hover:bg-primary-700 active:bg-primary-800'
                : 'border-[1.5px] border-primary-600 bg-white text-primary-700 hover:bg-primary-50'
            )}
          >
            {next.to === JobStatus.COMPLETED ? (
              <CheckCircle2 className="h-6 w-6" aria-hidden="true" />
            ) : (
              <Play className="h-6 w-6" aria-hidden="true" />
            )}
            {busy ? 'Saving…' : next.label}
          </button>
          {address && (
            <a
              href={mapsUrl(address)}
              target="_blank"
              rel="noopener noreferrer"
              aria-label={`Directions to ${visitTitle(job)}`}
              className="flex min-h-14 w-16 items-center justify-center rounded-xl border border-neutral-300 bg-white text-neutral-700 hover:bg-neutral-50"
            >
              <Navigation className="h-6 w-6" />
            </a>
          )}
        </div>
      )}
      {failed && (
        <p role="alert" className="mt-2 text-sm text-error-600">
          {getErrorMessage(action.error, "Couldn't update the visit")}
        </p>
      )}
    </li>
  );
}

export function MyDayPage() {
  const currentTenant = useTenantStore((s) => s.currentTenant);
  const timeZone = currentTenant?.settings?.timezone || undefined;
  const today = todayIn(timeZone);
  const [searchParams, setSearchParams] = useSearchParams();
  const requested = searchParams.get('date');
  const date = isDate(requested) ? requested : today;
  const action = useVisitAction();

  const { data, isLoading, isError, error } = useQuery({
    queryKey: myDayQueryKey(currentTenant?.id, date),
    queryFn: () => jobService.getMyDay(date),
    enabled: !!currentTenant,
  });

  const visits = data?.visits ?? [];
  const zone = data?.timezone || timeZone;
  const next = nextVisit(visits);
  const open = visits.filter((v) => !isDone(v));
  const starts = visits
    .map((v) => v.scheduled_start_at)
    .filter((t): t is number => typeof t === 'number');
  const ends = visits
    .map((v) => v.scheduled_end_at ?? v.scheduled_start_at)
    .filter((t): t is number => typeof t === 'number');

  const pickDay = (offset: number) => {
    const target = addDays(today, offset);
    setSearchParams(target === today ? {} : { date: target }, {
      replace: true,
    });
  };

  return (
    <DashboardLayout>
      <div className="mx-auto w-full max-w-xl px-4 pt-5 pb-10">
        {/* The top bar already carries the avatar. */}
        <header>
          <p className="text-neutral-500">{formatDayHeading(date)}</p>
          <h1 className="text-3xl font-bold tracking-tight text-neutral-900">
            My day
          </h1>
        </header>

        <div
          role="group"
          aria-label="Day"
          className="mt-4 grid grid-cols-3 gap-1 rounded-xl bg-neutral-100 p-1"
        >
          {DAYS.map(({ label, offset }) => {
            const active = addDays(today, offset) === date;
            return (
              <button
                key={label}
                type="button"
                onClick={() => pickDay(offset)}
                aria-pressed={active}
                className={cn(
                  'min-h-11 rounded-lg text-[15px] font-medium transition-colors',
                  active
                    ? 'bg-white text-neutral-900 shadow-sm'
                    : 'text-neutral-600 hover:text-neutral-900'
                )}
              >
                {label}
              </button>
            );
          })}
        </div>

        {visits.length > 0 && (
          <p className="mt-4 flex flex-wrap items-center gap-x-5 gap-y-1 text-neutral-600">
            <span className="flex items-center gap-1.5">
              <MapPin className="h-5 w-5 text-primary-600" aria-hidden="true" />
              {visits.length} {visits.length === 1 ? 'stop' : 'stops'}
            </span>
            {starts.length > 0 && (
              <span className="flex items-center gap-1.5">
                <Clock
                  className="h-5 w-5 text-primary-600"
                  aria-hidden="true"
                />
                {formatClock(Math.min(...starts), zone)} –{' '}
                {formatClock(Math.max(...ends), zone)}
              </span>
            )}
            <span className="flex items-center gap-1.5">
              <CalendarCheck
                className="h-5 w-5 text-primary-600"
                aria-hidden="true"
              />
              {open.length === 0 ? 'All done' : `${open.length} to go`}
            </span>
          </p>
        )}

        <div className="mt-5">
          {isLoading && (
            <p className="py-10 text-center text-neutral-500">
              Loading your visits…
            </p>
          )}
          {isError && (
            <p
              role="alert"
              className="rounded-xl border border-error-200 bg-error-50 px-4 py-3 text-error-700"
            >
              {getErrorMessage(error, "Couldn't load your day")}
            </p>
          )}
          {!isLoading && !isError && visits.length === 0 && (
            <div className="rounded-2xl border border-dashed border-neutral-300 bg-white px-6 py-10 text-center">
              <CalendarCheck className="mx-auto h-10 w-10 text-neutral-400" />
              <p className="mt-3 font-medium text-neutral-900">
                No visits {date === today ? 'today' : 'this day'}
              </p>
              <p className="mt-1 text-neutral-500">
                Visits the office assigns to you show up here.
              </p>
            </div>
          )}
          {visits.length > 0 && (
            <ul className="space-y-3" aria-label="Visits">
              {visits.map((job) => (
                <VisitCard
                  key={job.id}
                  job={job}
                  isNext={job.id === next?.id}
                  isToday={date === today}
                  timeZone={zone}
                  date={date}
                  action={action}
                />
              ))}
            </ul>
          )}
        </div>
      </div>
    </DashboardLayout>
  );
}
