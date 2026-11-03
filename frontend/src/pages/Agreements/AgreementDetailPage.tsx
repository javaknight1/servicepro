import { useNavigate, useParams } from 'react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  CalendarClock,
  CalendarRange,
  DollarSign,
  Loader2,
  Pause,
  Pencil,
  Play,
  Repeat,
  Trash2,
  UserRound,
} from 'lucide-react';
import { DashboardLayout } from '@components/layout';
import {
  Badge,
  Button,
  ErrorState,
  InfoCard,
  ListCard,
  PageHeader,
} from '@components/shared';
import {
  agreementKeys,
  agreementService,
  describeFrequency,
  monthlyAmount,
} from '@services/agreementService';
import { customerService } from '@services/customerService';
import { siteService, sitesQueryKey } from '@services/siteService';
import { tenantApi } from '@services/tenantService';
import { useTenantStore } from '@store/tenantStore';
import { formatDollars } from '@/utils/money';
import { getErrorMessage } from '@/utils/error';
import { customerName, formatClock, formatVisit } from './agreementDisplay';

const PREVIEW_DAYS = 14;

/** "Nov 2, 2026" from a YYYY-MM-DD date (no time zone shift). */
function formatDate(ymd: string): string {
  const [y, m, d] = ymd.split('-').map(Number);
  return new Date(Date.UTC(y, m - 1, d)).toLocaleDateString('en-US', {
    timeZone: 'UTC',
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

/** One service agreement: terms, pause/resume and the next visits (T167). */
export function AgreementDetailPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { id } = useParams<{ id: string }>();
  const tenantId = useTenantStore((s) => s.currentTenant?.id);

  const agreementQuery = useQuery({
    queryKey: agreementKeys.detail(id),
    queryFn: () => agreementService.get(id as string),
    enabled: !!id,
  });
  const agreement = agreementQuery.data;

  const customerQuery = useQuery({
    queryKey: ['customer', agreement?.customer_id ?? null],
    queryFn: () =>
      customerService.getCustomer(agreement?.customer_id as string),
    enabled: !!agreement,
  });
  const sitesQuery = useQuery({
    queryKey: sitesQueryKey(agreement?.customer_id, true),
    queryFn: () => siteService.list(agreement?.customer_id as string, true),
    enabled: !!agreement,
  });
  const membersQuery = useQuery({
    queryKey: ['tenant-members', tenantId ?? null],
    queryFn: async () =>
      (await tenantApi.getMembers(tenantId as string)).data.members ?? [],
    enabled: !!tenantId && !!agreement?.assigned_technician_id,
  });
  const previewQuery = useQuery({
    queryKey: agreementKeys.preview(id, PREVIEW_DAYS),
    queryFn: () => agreementService.preview(id as string, PREVIEW_DAYS),
    enabled: !!id && agreement?.status === 'active',
  });

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: agreementKeys.all });
  const toggle = useMutation({
    mutationFn: () =>
      agreement?.status === 'paused'
        ? agreementService.resume(id as string)
        : agreementService.pause(id as string),
    onSuccess: invalidate,
  });
  const remove = useMutation({
    mutationFn: () => agreementService.remove(id as string),
    onSuccess: async () => {
      await invalidate();
      navigate('/agreements');
    },
  });

  if (agreementQuery.isPending) {
    return (
      <DashboardLayout>
        <div className="flex min-h-96 items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
        </div>
      </DashboardLayout>
    );
  }
  if (agreementQuery.isError || !agreement) {
    return (
      <DashboardLayout
        breadcrumbs={[{ label: 'Agreements', to: '/agreements' }]}
      >
        <ErrorState
          title="Couldn't load this agreement"
          message={getErrorMessage(agreementQuery.error, 'Not found')}
          onRetry={() => void agreementQuery.refetch()}
        />
      </DashboardLayout>
    );
  }

  const site = sitesQuery.data?.find((s) => s.id === agreement.site_id);
  const customer = customerQuery.data;
  const tech = membersQuery.data?.find(
    (m) => m.user_id === agreement.assigned_technician_id
  );
  const techName = tech
    ? `${tech.first_name ?? ''} ${tech.last_name ?? ''}`.trim() || tech.email
    : agreement.assigned_technician_id
      ? '…'
      : 'Unassigned';
  const isPaused = agreement.status === 'paused';
  const title = site?.name ?? agreement.title;
  const actionError = toggle.error ?? remove.error;

  return (
    <DashboardLayout
      breadcrumbs={[
        { label: 'Agreements', to: '/agreements' },
        { label: title },
      ]}
    >
      <div className="mx-auto max-w-[1200px] p-4 sm:p-6 lg:p-8">
        <PageHeader
          title={title}
          badge={
            <Badge size="sm" variant={isPaused ? 'neutral' : 'success'}>
              {isPaused ? 'Paused' : 'Active'}
            </Badge>
          }
          subtitle={`${agreement.title} · ${customerName(customer)}`}
          actions={
            <>
              <Button
                variant="secondary"
                onClick={() => {
                  if (
                    window.confirm(
                      'Delete this agreement? Visits already scheduled stay on the schedule.'
                    )
                  ) {
                    remove.mutate();
                  }
                }}
                isLoading={remove.isPending}
                className="gap-2"
              >
                <Trash2 className="h-4 w-4" />
                Delete
              </Button>
              <Button
                variant="outline"
                onClick={() => toggle.mutate()}
                isLoading={toggle.isPending}
                className="gap-2"
              >
                {isPaused ? (
                  <Play className="h-4 w-4" />
                ) : (
                  <Pause className="h-4 w-4" />
                )}
                {isPaused ? 'Resume' : 'Pause'}
              </Button>
              <Button
                onClick={() => navigate(`/agreements/${agreement.id}/edit`)}
                className="gap-2"
              >
                <Pencil className="h-4 w-4" />
                Edit
              </Button>
            </>
          }
        />
        {actionError && (
          <p
            role="alert"
            className="mb-6 rounded-lg border border-error-200 bg-error-50 p-4 text-error-700"
          >
            {getErrorMessage(actionError, 'Could not update the agreement')}
          </p>
        )}

        <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <InfoCard
            icon={<Repeat />}
            label="Frequency"
            value={describeFrequency(agreement)}
            detail={`${formatClock(agreement.visit_start_time)} · ${agreement.visit_duration_minutes} min visits`}
          />
          <InfoCard
            icon={<DollarSign />}
            label="Price"
            value={`${formatDollars(agreement.price_per_period)}/${agreement.billing_period}`}
            detail={[
              agreement.billing_period === 'visit'
                ? `About ${formatDollars(monthlyAmount(agreement), { wholeUnits: true })}/mo`
                : 'Before sales tax',
              agreement.po_number ? `PO ${agreement.po_number}` : '',
            ]
              .filter(Boolean)
              .join(' · ')}
          />
          <InfoCard
            icon={<CalendarRange />}
            label="Term"
            value={`From ${formatDate(agreement.start_date)}`}
            detail={
              agreement.end_date
                ? `Until ${formatDate(agreement.end_date)}`
                : 'No end date'
            }
          />
          <InfoCard
            icon={<UserRound />}
            label="Lead tech"
            value={techName}
            detail={site?.address_full}
          />
        </div>

        <ListCard
          title={`Next ${PREVIEW_DAYS} days`}
          count={previewQuery.data?.visits.length}
          bodyClassName="px-6 pb-6"
        >
          {isPaused ? (
            <p className="py-4 text-sm text-neutral-500">
              Paused: no visits are scheduled until you resume it.
            </p>
          ) : previewQuery.isPending ? (
            <Loader2 className="h-5 w-5 animate-spin text-neutral-400" />
          ) : previewQuery.isError ? (
            <p className="py-4 text-sm text-error-700">
              {getErrorMessage(previewQuery.error, 'Could not load visits')}
            </p>
          ) : previewQuery.data.visits.length === 0 ? (
            <p className="py-4 text-sm text-neutral-500">
              No visits in the next {PREVIEW_DAYS} days.
            </p>
          ) : (
            <ul
              className="divide-y divide-neutral-200"
              aria-label="Upcoming visits"
            >
              {previewQuery.data.visits.map((v) => (
                <li key={v.date} className="flex items-center gap-3 py-3">
                  <CalendarClock className="h-5 w-5 shrink-0 text-primary-600" />
                  <span className="text-[15px] text-neutral-900">
                    {formatVisit(v.starts_at, previewQuery.data.timezone)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </ListCard>

        {agreement.notes && (
          <section className="mt-6 rounded-2xl border border-neutral-200 bg-white p-6">
            <h2 className="mb-2 text-lg font-semibold text-neutral-900">
              Notes
            </h2>
            <p className="whitespace-pre-line text-neutral-700">
              {agreement.notes}
            </p>
          </section>
        )}
      </div>
    </DashboardLayout>
  );
}
