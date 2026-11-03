/**
 * One visit on a phone (T173, mockup 16 where it applies today): what to do,
 * how to get in, notes and materials used, and the big Start / Complete
 * action. Checklists (T175), photos (T174) and the sign-off (T176) slot in
 * here later. Never shows money.
 */
import { useRef, useState, type FormEvent } from 'react';
import { Link, useParams, useSearchParams } from 'react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { isAxiosError } from 'axios';
import {
  CheckCircle2,
  ChevronLeft,
  MapPin,
  Package,
  PenLine,
  Phone,
  Play,
} from 'lucide-react';
import { DashboardLayout } from '@components/layout';
import { useTenantStore } from '@store';
import { jobService, JobStatus, type Job } from '@services/jobService';
import { cn } from '@utils/cn';
import { getErrorMessage } from '@/utils/error';
import {
  formatAddress,
  formatClock,
  isDate,
  mapsUrl,
  primaryAction,
  telUrl,
  visitAddress,
  visitContact,
  visitQueryKey,
  visitStatusLabel,
  visitTitle,
  customerSubtitle,
} from './myDay';
import { useVisitAction } from './useVisitAction';

const inputClass =
  'w-full rounded-xl border border-neutral-300 bg-white px-3 py-3 text-base text-neutral-900 placeholder:text-neutral-400 focus:border-primary-500 focus:ring-2 focus:ring-primary-200 focus:outline-hidden';

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-2xl border border-neutral-200 bg-white p-4">
      <h2 className="text-lg font-semibold text-neutral-900">{title}</h2>
      <div className="mt-2">{children}</div>
    </section>
  );
}

function NotesSection({
  job,
  noteRef,
}: {
  job: Job;
  noteRef: React.RefObject<HTMLTextAreaElement | null>;
}) {
  const queryClient = useQueryClient();
  const [note, setNote] = useState('');
  const add = useMutation({
    mutationFn: (text: string) => jobService.addNote(job.id, text),
    onSuccess: () => {
      setNote('');
      queryClient.invalidateQueries({ queryKey: visitQueryKey(job.id) });
    },
  });
  const notes = [...(job.notes ?? [])].sort((a, b) =>
    b.created_at.localeCompare(a.created_at)
  );
  const submit = (e: FormEvent) => {
    e.preventDefault();
    const text = note.trim();
    if (text) add.mutate(text);
  };

  return (
    <Section title="Notes">
      <form onSubmit={submit} className="space-y-2">
        <label htmlFor="visit-note" className="sr-only">
          Add a note
        </label>
        <textarea
          id="visit-note"
          ref={noteRef}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          rows={3}
          placeholder="What did you find or do?"
          className={inputClass}
        />
        <button
          type="submit"
          disabled={!note.trim() || add.isPending}
          className="min-h-12 w-full rounded-xl border-[1.5px] border-primary-600 bg-white text-base font-semibold text-primary-700 hover:bg-primary-50 disabled:opacity-50"
        >
          {add.isPending ? 'Saving…' : 'Save note'}
        </button>
        {add.isError && (
          <p role="alert" className="text-sm text-error-600">
            {getErrorMessage(add.error, "Couldn't save the note")}
          </p>
        )}
      </form>
      {notes.length > 0 ? (
        <ul className="mt-4 divide-y divide-neutral-100" aria-label="Notes">
          {notes.map((n) => (
            <li key={n.id} className="py-3">
              <p className="whitespace-pre-wrap text-neutral-900">{n.note}</p>
              <p className="mt-1 text-sm text-neutral-500">
                {[
                  n.user_name,
                  new Date(n.created_at).toLocaleString('en-US', {
                    month: 'short',
                    day: 'numeric',
                    hour: 'numeric',
                    minute: '2-digit',
                  }),
                ]
                  .filter(Boolean)
                  .join(' · ')}
              </p>
            </li>
          ))}
        </ul>
      ) : (
        <p className="mt-3 text-neutral-500">No notes yet.</p>
      )}
    </Section>
  );
}

function MaterialsSection({ job }: { job: Job }) {
  const queryClient = useQueryClient();
  const [name, setName] = useState('');
  const [quantity, setQuantity] = useState('1');
  const [unit, setUnit] = useState('each');
  const add = useMutation({
    mutationFn: () =>
      jobService.addMaterial(job.id, {
        name: name.trim(),
        quantity: Number(quantity),
        unit: unit.trim() || 'each',
      }),
    onSuccess: () => {
      setName('');
      setQuantity('1');
      queryClient.invalidateQueries({ queryKey: visitQueryKey(job.id) });
    },
  });
  const qty = Number(quantity);
  const valid = name.trim() !== '' && Number.isFinite(qty) && qty > 0;
  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (valid) add.mutate();
  };
  const materials = job.materials ?? [];

  return (
    <Section title="Materials used">
      {materials.length > 0 ? (
        <ul className="mb-3 divide-y divide-neutral-100" aria-label="Materials">
          {materials.map((m) => (
            <li
              key={m.id}
              className="flex items-center justify-between gap-3 py-2.5"
            >
              <span className="flex items-center gap-2 text-neutral-900">
                <Package
                  className="h-4 w-4 text-neutral-400"
                  aria-hidden="true"
                />
                {m.name}
              </span>
              <span className="shrink-0 text-neutral-600">
                {m.quantity} {m.unit}
              </span>
            </li>
          ))}
        </ul>
      ) : (
        <p className="mb-3 text-neutral-500">Nothing logged yet.</p>
      )}
      <form onSubmit={submit} className="space-y-2">
        <label className="block">
          <span className="text-sm font-medium text-neutral-700">Material</span>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Trash liners 33 gal"
            className={cn(inputClass, 'mt-1')}
          />
        </label>
        <div className="grid grid-cols-2 gap-2">
          <label className="block">
            <span className="text-sm font-medium text-neutral-700">
              Quantity
            </span>
            <input
              value={quantity}
              onChange={(e) => setQuantity(e.target.value)}
              inputMode="decimal"
              className={cn(inputClass, 'mt-1')}
            />
          </label>
          <label className="block">
            <span className="text-sm font-medium text-neutral-700">Unit</span>
            <input
              value={unit}
              onChange={(e) => setUnit(e.target.value)}
              className={cn(inputClass, 'mt-1')}
            />
          </label>
        </div>
        <button
          type="submit"
          disabled={!valid || add.isPending}
          className="min-h-12 w-full rounded-xl border-[1.5px] border-primary-600 bg-white text-base font-semibold text-primary-700 hover:bg-primary-50 disabled:opacity-50"
        >
          {add.isPending ? 'Saving…' : 'Add material'}
        </button>
        {add.isError && (
          <p role="alert" className="text-sm text-error-600">
            {getErrorMessage(add.error, "Couldn't add the material")}
          </p>
        )}
      </form>
    </Section>
  );
}

export function VisitPage() {
  const { jobId } = useParams<{ jobId: string }>();
  const [searchParams] = useSearchParams();
  const day = searchParams.get('date');
  const backPath = isDate(day) ? `/my-day?date=${day}` : '/my-day';
  const currentTenant = useTenantStore((s) => s.currentTenant);
  const timeZone = currentTenant?.settings?.timezone || undefined;
  const action = useVisitAction();
  const noteRef = useRef<HTMLTextAreaElement>(null);

  const {
    data: job,
    isLoading,
    error,
  } = useQuery({
    queryKey: visitQueryKey(jobId),
    queryFn: () => jobService.getJob(jobId as string),
    enabled: !!jobId,
    retry: (count, err) =>
      !(isAxiosError(err) && err.response?.status === 404) && count < 2,
  });

  const back = (
    <Link
      to={backPath}
      className="inline-flex min-h-11 items-center gap-1 text-lg font-medium text-primary-600"
    >
      <ChevronLeft className="h-5 w-5" aria-hidden="true" />
      My day
    </Link>
  );

  if (isLoading || !job) {
    const notFound = isAxiosError(error) && error.response?.status === 404;
    return (
      <DashboardLayout>
        <div className="mx-auto w-full max-w-xl px-4 pt-4 pb-10">
          {back}
          {isLoading ? (
            <p className="py-10 text-center text-neutral-500">Loading visit…</p>
          ) : (
            <p
              role="alert"
              className="mt-4 rounded-xl border border-error-200 bg-error-50 px-4 py-3 text-error-700"
            >
              {notFound
                ? 'Visit not found. It may have been reassigned.'
                : getErrorMessage(error, "Couldn't load the visit")}
            </p>
          )}
        </div>
      </DashboardLayout>
    );
  }

  const next = primaryAction(job);
  const address = visitAddress(job);
  const contact = visitContact(job);
  const callUrl = contact ? telUrl(contact.phone) : null;
  const customer = customerSubtitle(job);
  const crew = (job.assignments ?? [])
    .filter((a) => !a.unassigned_at && a.user_name)
    .map((a) => a.user_name);
  const meta = [
    job.job_number,
    job.actual_start_at
      ? `Started ${formatClock(job.actual_start_at, timeZone)}`
      : job.scheduled_start_at
        ? `Scheduled ${formatClock(job.scheduled_start_at, timeZone)}`
        : null,
    crew.join(', ') || null,
  ].filter(Boolean);
  const busy = action.pendingJobId === job.id;

  return (
    <DashboardLayout>
      <div className="mx-auto w-full max-w-xl px-4 pt-4 pb-10">
        <div className="flex items-center justify-between gap-3">
          {back}
          <span
            className={cn(
              'rounded-full border px-3 py-1 text-sm font-medium',
              job.status === JobStatus.IN_PROGRESS
                ? 'border-warning-200 bg-warning-50 text-warning-700'
                : 'border-neutral-200 bg-white text-neutral-700'
            )}
          >
            {visitStatusLabel(job.status)}
          </span>
        </div>
        <h1 className="mt-2 text-3xl font-bold leading-tight tracking-tight text-neutral-900">
          {visitTitle(job)}
        </h1>
        {customer && <p className="mt-1 text-neutral-600">{customer}</p>}
        <p className="mt-1 text-neutral-600">
          {meta.map((part, i) => (
            <span key={i}>
              {i > 0 && ' · '}
              <span className={i === 0 ? 'text-primary-600' : undefined}>
                {part}
              </span>
            </span>
          ))}
        </p>

        {job.actual_end_at && (
          <p className="mt-3 flex items-center gap-2 font-medium text-success-700">
            <CheckCircle2 className="h-5 w-5" aria-hidden="true" />
            Completed {formatClock(job.actual_end_at, timeZone)}
          </p>
        )}

        <div className="mt-5 space-y-3">
          <Section title="What to do">
            <p className="font-medium text-neutral-900">{job.title}</p>
            {job.description ? (
              <p className="mt-1 whitespace-pre-wrap text-neutral-700">
                {job.description}
              </p>
            ) : (
              <p className="mt-1 text-neutral-500">No instructions added.</p>
            )}
            {job.special_instructions && (
              <p className="mt-2 whitespace-pre-wrap text-neutral-700">
                {job.special_instructions}
              </p>
            )}
          </Section>

          {(address || job.site?.access_notes || contact) && (
            <Section title="Site">
              {address && (
                <a
                  href={mapsUrl(address)}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex min-h-11 items-center gap-2 text-neutral-800 hover:text-primary-700"
                >
                  <MapPin className="h-5 w-5 shrink-0 text-neutral-400" />
                  <span className="underline decoration-neutral-300 underline-offset-2">
                    {formatAddress(address)}
                  </span>
                </a>
              )}
              {job.site?.access_notes && (
                <p className="mt-2 rounded-xl border border-warning-200 bg-warning-50 px-3 py-2 text-warning-800">
                  Site note: {job.site.access_notes}
                </p>
              )}
              {contact && callUrl && (
                <a
                  href={callUrl}
                  className="mt-2 flex min-h-11 items-center gap-2 text-neutral-800 hover:text-primary-700"
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
            </Section>
          )}

          <NotesSection job={job} noteRef={noteRef} />
          <MaterialsSection job={job} />
        </div>

        <div className="sticky bottom-0 -mx-4 mt-5 border-t border-neutral-200 bg-white/95 px-4 pt-3 pb-4 backdrop-blur-sm">
          <div className="flex gap-3">
            <button
              type="button"
              onClick={() => noteRef.current?.focus()}
              className="flex min-h-14 flex-1 items-center justify-center gap-2 rounded-xl border border-neutral-300 bg-white text-lg font-semibold text-neutral-800 hover:bg-neutral-50"
            >
              <PenLine className="h-5 w-5" aria-hidden="true" />
              Add note
            </button>
            {next && (
              <button
                type="button"
                onClick={() => action.run(job, next)}
                disabled={busy}
                className="flex min-h-14 flex-1 items-center justify-center gap-2 rounded-xl bg-primary-600 text-lg font-semibold text-white hover:bg-primary-700 active:bg-primary-800 disabled:opacity-60"
              >
                {next.to === JobStatus.COMPLETED ? (
                  <CheckCircle2 className="h-6 w-6" aria-hidden="true" />
                ) : (
                  <Play className="h-6 w-6" aria-hidden="true" />
                )}
                {busy ? 'Saving…' : next.label}
              </button>
            )}
          </div>
          {action.failedJobId === job.id && (
            <p role="alert" className="mt-2 text-sm text-error-600">
              {getErrorMessage(action.error, "Couldn't update the visit")}
            </p>
          )}
        </div>
      </div>
    </DashboardLayout>
  );
}
