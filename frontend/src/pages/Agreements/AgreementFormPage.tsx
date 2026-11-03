import { useEffect, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Loader2 } from 'lucide-react';
import { DashboardLayout } from '@components/layout';
import { Button, Input, PageHeader } from '@components/shared';
import {
  agreementKeys,
  agreementService,
  type Agreement,
  type AgreementFrequency,
  type AgreementSchedule,
  type BillingPeriod,
} from '@services/agreementService';
import { fetchAllCustomers } from '@services/customerLookup';
import { siteService, sitesQueryKey } from '@services/siteService';
import { tenantApi } from '@services/tenantService';
import { JobType } from '@services/jobService';
import { useTenantStore } from '@store/tenantStore';
import { getErrorMessage, getFieldErrors } from '@/utils/error';
import { cn } from '@utils/cn';
import { customerName } from './agreementDisplay';

interface AgreementForm {
  customer_id: string;
  site_id: string;
  title: string;
  frequency: AgreementFrequency;
  weekdays: number[];
  times_per_week: string;
  day_of_month: string;
  start_date: string;
  end_date: string;
  visit_start_time: string;
  visit_duration_minutes: string;
  price_per_period: string;
  billing_period: BillingPeriod;
  assigned_technician_id: string;
  job_type: JobType;
  notes: string;
}

function today(): string {
  return new Date().toLocaleDateString('en-CA');
}

const emptyForm = (): AgreementForm => ({
  customer_id: '',
  site_id: '',
  title: '',
  frequency: 'weekdays',
  weekdays: [],
  times_per_week: '3',
  day_of_month: '1',
  start_date: today(),
  end_date: '',
  visit_start_time: '18:00',
  visit_duration_minutes: '120',
  price_per_period: '',
  billing_period: 'month',
  assigned_technician_id: '',
  job_type: JobType.MAINTENANCE,
  notes: '',
});

function toForm(a: Agreement): AgreementForm {
  return {
    customer_id: a.customer_id,
    site_id: a.site_id,
    title: a.title,
    frequency: a.frequency,
    weekdays: a.weekdays ?? [],
    times_per_week: String(a.times_per_week ?? 3),
    day_of_month: String(a.day_of_month ?? 1),
    start_date: a.start_date,
    end_date: a.end_date ?? '',
    visit_start_time: a.visit_start_time,
    visit_duration_minutes: String(a.visit_duration_minutes),
    price_per_period: a.price_per_period,
    billing_period: a.billing_period,
    assigned_technician_id: a.assigned_technician_id ?? '',
    job_type: a.job_type,
    notes: a.notes ?? '',
  };
}

/** The frequency fields the API needs for the chosen frequency. */
function toSchedule(form: AgreementForm): AgreementSchedule {
  switch (form.frequency) {
    case 'weekdays':
      return { frequency: 'weekdays' };
    case 'weekly':
      return { frequency: 'weekly', weekdays: form.weekdays };
    case 'times_per_week':
      return {
        frequency: 'times_per_week',
        times_per_week: Number(form.times_per_week),
        // Specific days only when exactly that many are picked.
        ...(form.weekdays.length === Number(form.times_per_week)
          ? { weekdays: form.weekdays }
          : {}),
      };
    case 'monthly':
      return { frequency: 'monthly', day_of_month: Number(form.day_of_month) };
  }
}

// Monday first, as on the schedule.
const WEEKDAYS: Array<[number, string]> = [
  [1, 'Mon'],
  [2, 'Tue'],
  [3, 'Wed'],
  [4, 'Thu'],
  [5, 'Fri'],
  [6, 'Sat'],
  [0, 'Sun'],
];

const selectClass =
  'w-full rounded-lg border border-neutral-300 px-3 py-2 focus:border-primary-500 focus:ring-2 focus:ring-primary-500 disabled:bg-neutral-100';
const labelClass = 'mb-1.5 block text-sm font-medium text-neutral-700';

/** Create or edit a service agreement (T167). */
export function AgreementFormPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const location = useLocation();
  const { id } = useParams<{ id: string }>();
  const isNew = !id;
  const prefill = location.state as {
    customerId?: string;
    siteId?: string;
  } | null;
  const tenantId = useTenantStore((s) => s.currentTenant?.id);

  const [form, setForm] = useState<AgreementForm>(() => ({
    ...emptyForm(),
    customer_id: prefill?.customerId ?? '',
    site_id: prefill?.siteId ?? '',
  }));

  const agreementQuery = useQuery({
    queryKey: agreementKeys.detail(id),
    queryFn: () => agreementService.get(id as string),
    enabled: !isNew,
  });
  const loaded = agreementQuery.data;
  useEffect(() => {
    if (loaded) setForm(toForm(loaded));
  }, [loaded]);

  const customersQuery = useQuery({
    queryKey: ['customers', 'all'],
    queryFn: () => fetchAllCustomers(),
  });
  const sitesQuery = useQuery({
    queryKey: sitesQueryKey(form.customer_id, true),
    queryFn: () => siteService.list(form.customer_id, true),
    enabled: !!form.customer_id,
  });
  const membersQuery = useQuery({
    queryKey: ['tenant-members', tenantId ?? null],
    queryFn: async () =>
      (await tenantApi.getMembers(tenantId as string)).data.members ?? [],
    enabled: !!tenantId,
  });

  const save = useMutation({
    mutationFn: async (f: AgreementForm): Promise<Agreement> => {
      const common = {
        title: f.title.trim(),
        start_date: f.start_date,
        visit_start_time: f.visit_start_time,
        visit_duration_minutes: Number(f.visit_duration_minutes) || undefined,
        price_per_period: f.price_per_period.trim(),
        billing_period: f.billing_period,
        job_type: f.job_type,
        notes: f.notes.trim(),
      };
      if (isNew) {
        return agreementService.create({
          ...common,
          ...toSchedule(f),
          customer_id: f.customer_id,
          site_id: f.site_id,
          end_date: f.end_date || undefined,
          assigned_technician_id: f.assigned_technician_id || undefined,
          notes: common.notes || undefined,
        });
      }
      return agreementService.update(id as string, {
        ...common,
        schedule: toSchedule(f),
        ...(f.end_date ? { end_date: f.end_date } : { clear_end_date: true }),
        ...(f.assigned_technician_id
          ? { assigned_technician_id: f.assigned_technician_id }
          : { clear_technician: true }),
      });
    },
    onSuccess: async (saved) => {
      await queryClient.invalidateQueries({ queryKey: agreementKeys.all });
      navigate(`/agreements/${saved.id}`);
    },
  });

  const fieldErrors = save.error ? getFieldErrors(save.error) : {};
  const set =
    <K extends keyof AgreementForm>(name: K) =>
    (
      e: React.ChangeEvent<
        HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement
      >
    ) => {
      const { value } = e.target;
      setForm((prev) => ({ ...prev, [name]: value }));
    };
  const toggleDay = (day: number) =>
    setForm((prev) => ({
      ...prev,
      weekdays: prev.weekdays.includes(day)
        ? prev.weekdays.filter((d) => d !== day)
        : [...prev.weekdays, day],
    }));

  const activeSites = (sitesQuery.data ?? []).filter(
    (s) => s.is_active || s.id === form.site_id
  );
  const title = isNew ? 'New agreement' : 'Edit agreement';

  if (!isNew && agreementQuery.isPending) {
    return (
      <DashboardLayout>
        <div className="flex min-h-96 items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout
      breadcrumbs={[
        { label: 'Agreements', to: '/agreements' },
        { label: title },
      ]}
    >
      <div className="mx-auto max-w-4xl p-4 sm:p-6 lg:p-8">
        <PageHeader
          title={title}
          subtitle="Recurring work at one site: how often, when, and what it costs."
        />
        {(save.error || agreementQuery.error) && (
          <p
            role="alert"
            className="mb-6 rounded-lg border border-error-200 bg-error-50 p-4 text-error-700"
          >
            {getErrorMessage(
              save.error ?? agreementQuery.error,
              'Could not save the agreement'
            )}
          </p>
        )}
        <form
          onSubmit={(e) => {
            e.preventDefault();
            save.mutate(form);
          }}
          className="space-y-6"
        >
          <section className="space-y-4 rounded-2xl border border-neutral-200 bg-white p-6">
            <h2 className="text-lg font-semibold text-neutral-900">
              Customer and site
            </h2>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div>
                <label htmlFor="customer_id" className={labelClass}>
                  Customer *
                </label>
                <select
                  id="customer_id"
                  value={form.customer_id}
                  onChange={(e) =>
                    setForm((p) => ({
                      ...p,
                      customer_id: e.target.value,
                      site_id: '',
                    }))
                  }
                  required
                  disabled={!isNew || customersQuery.isPending}
                  className={selectClass}
                >
                  <option value="">Select a customer</option>
                  {(customersQuery.data ?? []).map((c) => (
                    <option key={c.id} value={c.id}>
                      {customerName(c)}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label htmlFor="site_id" className={labelClass}>
                  Site *
                </label>
                <select
                  id="site_id"
                  value={form.site_id}
                  onChange={set('site_id')}
                  required
                  disabled={!isNew || !form.customer_id}
                  className={selectClass}
                >
                  <option value="">
                    {form.customer_id && !sitesQuery.isPending
                      ? activeSites.length === 0
                        ? 'This customer has no sites'
                        : 'Select a site'
                      : 'Select a customer first'}
                  </option>
                  {activeSites.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
                </select>
                {fieldErrors.site_id && (
                  <p className="mt-1.5 text-sm text-error-600">
                    {fieldErrors.site_id}
                  </p>
                )}
              </div>
            </div>
            <Input
              id="title"
              label="Title *"
              value={form.title}
              onChange={set('title')}
              required
              maxLength={200}
              placeholder="Nightly cleaning"
              error={fieldErrors.title}
              fullWidth
            />
          </section>

          <section className="space-y-4 rounded-2xl border border-neutral-200 bg-white p-6">
            <h2 className="text-lg font-semibold text-neutral-900">Schedule</h2>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div>
                <label htmlFor="frequency" className={labelClass}>
                  Frequency *
                </label>
                <select
                  id="frequency"
                  value={form.frequency}
                  onChange={set('frequency')}
                  className={selectClass}
                >
                  <option value="weekdays">Every weekday (Mon–Fri)</option>
                  <option value="weekly">Weekly on chosen days</option>
                  <option value="times_per_week">Times per week</option>
                  <option value="monthly">Monthly</option>
                </select>
              </div>
              {form.frequency === 'times_per_week' && (
                <Input
                  id="times_per_week"
                  label="Times per week *"
                  type="number"
                  min={1}
                  max={7}
                  value={form.times_per_week}
                  onChange={set('times_per_week')}
                  required
                  error={fieldErrors.times_per_week}
                  fullWidth
                />
              )}
              {form.frequency === 'monthly' && (
                <Input
                  id="day_of_month"
                  label="Day of the month *"
                  type="number"
                  min={1}
                  max={31}
                  value={form.day_of_month}
                  onChange={set('day_of_month')}
                  required
                  helperText="Days 29–31 fall on the last day in shorter months."
                  error={fieldErrors.day_of_month}
                  fullWidth
                />
              )}
            </div>
            {(form.frequency === 'weekly' ||
              form.frequency === 'times_per_week') && (
              <fieldset>
                <legend className={labelClass}>
                  {form.frequency === 'weekly'
                    ? 'Days *'
                    : 'Days (optional: pick exactly that many)'}
                </legend>
                <div className="flex flex-wrap gap-2">
                  {WEEKDAYS.map(([day, label]) => {
                    const on = form.weekdays.includes(day);
                    return (
                      <button
                        key={day}
                        type="button"
                        aria-pressed={on}
                        onClick={() => toggleDay(day)}
                        className={cn(
                          'rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors',
                          on
                            ? 'border-primary-600 bg-primary-50 text-primary-700'
                            : 'border-neutral-300 text-neutral-700 hover:bg-neutral-50'
                        )}
                      >
                        {label}
                      </button>
                    );
                  })}
                </div>
                {fieldErrors.weekdays && (
                  <p className="mt-1.5 text-sm text-error-600">
                    {fieldErrors.weekdays}
                  </p>
                )}
              </fieldset>
            )}
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <Input
                id="start_date"
                label="Starts *"
                type="date"
                value={form.start_date}
                onChange={set('start_date')}
                required
                error={fieldErrors.start_date}
                fullWidth
              />
              <Input
                id="end_date"
                label="Ends"
                type="date"
                value={form.end_date}
                onChange={set('end_date')}
                helperText="Leave empty to keep it running."
                error={fieldErrors.end_date}
                fullWidth
              />
              <Input
                id="visit_start_time"
                label="Visit start time *"
                type="time"
                value={form.visit_start_time}
                onChange={set('visit_start_time')}
                required
                error={fieldErrors.visit_start_time}
                fullWidth
              />
              <Input
                id="visit_duration_minutes"
                label="Visit length (minutes)"
                type="number"
                min={1}
                max={1440}
                value={form.visit_duration_minutes}
                onChange={set('visit_duration_minutes')}
                error={fieldErrors.visit_duration_minutes}
                fullWidth
              />
            </div>
          </section>

          <section className="space-y-4 rounded-2xl border border-neutral-200 bg-white p-6">
            <h2 className="text-lg font-semibold text-neutral-900">
              Price and crew
            </h2>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <Input
                id="price_per_period"
                label="Price ($) *"
                type="number"
                inputMode="decimal"
                min="0"
                step="0.01"
                value={form.price_per_period}
                onChange={set('price_per_period')}
                required
                error={fieldErrors.price_per_period}
                fullWidth
              />
              <div>
                <label htmlFor="billing_period" className={labelClass}>
                  Price is per
                </label>
                <select
                  id="billing_period"
                  value={form.billing_period}
                  onChange={set('billing_period')}
                  className={selectClass}
                >
                  <option value="month">Month</option>
                  <option value="visit">Visit</option>
                </select>
              </div>
              <div>
                <label htmlFor="assigned_technician_id" className={labelClass}>
                  Lead tech
                </label>
                <select
                  id="assigned_technician_id"
                  value={form.assigned_technician_id}
                  onChange={set('assigned_technician_id')}
                  className={selectClass}
                >
                  <option value="">Unassigned</option>
                  {(membersQuery.data ?? [])
                    .filter((m) => m.is_active)
                    .map((m) => (
                      <option key={m.user_id} value={m.user_id}>
                        {`${m.first_name ?? ''} ${m.last_name ?? ''}`.trim() ||
                          m.email}
                      </option>
                    ))}
                </select>
              </div>
              <div>
                <label htmlFor="job_type" className={labelClass}>
                  Visit job type
                </label>
                <select
                  id="job_type"
                  value={form.job_type}
                  onChange={set('job_type')}
                  className={selectClass}
                >
                  <option value={JobType.MAINTENANCE}>Maintenance</option>
                  <option value={JobType.INSPECTION}>Inspection</option>
                  <option value={JobType.REPAIR}>Repair</option>
                  <option value={JobType.INSTALLATION}>Installation</option>
                </select>
              </div>
            </div>
            <div>
              <label htmlFor="notes" className={labelClass}>
                Notes
              </label>
              <textarea
                id="notes"
                value={form.notes}
                onChange={set('notes')}
                rows={3}
                maxLength={5000}
                className={selectClass}
              />
            </div>
          </section>

          <div className="flex justify-end gap-3">
            <Button
              type="button"
              variant="secondary"
              onClick={() =>
                navigate(isNew ? '/agreements' : `/agreements/${id}`)
              }
            >
              Cancel
            </Button>
            <Button type="submit" isLoading={save.isPending}>
              {isNew ? 'Create agreement' : 'Save changes'}
            </Button>
          </div>
        </form>
      </div>
    </DashboardLayout>
  );
}
