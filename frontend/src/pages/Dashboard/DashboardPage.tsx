import { useState, useEffect, useCallback } from 'react';
import { Link, Navigate, useNavigate } from 'react-router';
import { DashboardLayout } from '@components/layout';
import {
  Card,
  CardContent,
  StatsCard,
  Button,
  PageHeader,
  ListCard,
  ListCardRow,
} from '@components/shared';
import { useAuthStore, useTenantStore } from '@store';
import { jobService, JobStats, Job, JobStatus } from '@services/jobService';
import { invoiceService, InvoiceStats } from '@services/invoiceService';
import { arAgingService } from '@services/arAgingService';
import type { ARAgingBucketSummary, AgingBucket } from '@app-types/arAging';
import {
  AGING_BUCKETS,
  AGING_BUCKET_STYLES,
} from '@components/reports/agingStyles';
import {
  Users,
  Briefcase,
  FileText,
  Receipt,
  Building2,
  Rocket,
  Plus,
  Hourglass,
  HandCoins,
  Wrench,
} from 'lucide-react';
import { formatDollars, toAmount } from '@/utils/money';
import { isValidTimeZone } from '@/utils/epoch';
import { cn } from '@utils/cn';
import {
  APP_PERMISSIONS,
  isFieldUser,
  useTenantPermissions,
} from '@/utils/tenantPermissions';
import { useInvoiceJob } from '@/hooks/useInvoiceJob';

/** Completed jobs that have no invoice yet (creating one moves a job to "invoiced"). */
const JOBS_TO_INVOICE_SHOWN = 6;

interface JobsToInvoice {
  jobs: Job[];
  total: number;
  /** When the longest-waiting job was finished (epoch seconds). */
  oldestDoneAt?: number;
}

interface DashboardData {
  jobs: JobStats | null;
  invoices: InvoiceStats | null;
  toInvoice: JobsToInvoice | null;
  aging: ARAgingBucketSummary[] | null;
  /** A technician's own jobs, soonest first (T035; the API lists only theirs). */
  myJobs: Job[] | null;
}

const emptyData: DashboardData = {
  jobs: null,
  invoices: null,
  toInvoice: null,
  aging: null,
  myJobs: null,
};

const OVER_30_DAYS: AgingBucket[] = ['31-60', '61-90', '90+'];

function customerName(job: Job): string {
  const c = job.customer;
  if (!c) return 'Customer';
  return c.company_name || `${c.first_name} ${c.last_name}`.trim() || c.email;
}

/** What the job is worth, for the invoice: its total, actual or estimated cost. */
function jobAmount(job: Job): number | undefined {
  return [job.total_amount, job.actual_cost, job.estimated_cost]
    .map((v) => toAmount(v))
    .find((v) => v > 0);
}

function sumBuckets(
  buckets: ARAgingBucketSummary[],
  which: AgingBucket[]
): number {
  return buckets
    .filter((b) => which.includes(b.bucket))
    .reduce((sum, b) => sum + toAmount(b.total_amount), 0);
}

function useDateFormatters(timeZone: string | undefined) {
  const tz = isValidTimeZone(timeZone) ? timeZone : undefined;
  const longDate = new Intl.DateTimeFormat('en-US', {
    weekday: 'long',
    month: 'long',
    day: 'numeric',
    year: 'numeric',
    timeZone: tz,
  });
  const shortDate = new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    timeZone: tz,
  });
  const month = new Intl.DateTimeFormat('en-US', {
    month: 'short',
    timeZone: tz,
  });
  return {
    today: longDate.format(new Date()),
    thisMonth: month.format(new Date()),
    shortEpoch: (epoch: number) => shortDate.format(new Date(epoch * 1000)),
  };
}

export function DashboardPage() {
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const {
    currentTenant,
    tenants,
    isLoading: tenantsLoading,
    loadTenants,
  } = useTenantStore();
  const [data, setData] = useState<DashboardData>(emptyData);
  const [isLoading, setIsLoading] = useState(true);
  const [tenantsChecked, setTenantsChecked] = useState(false);
  const dates = useDateFormatters(currentTenant?.settings?.timezone);
  // Money (A/R, collections, invoicing) only for callers who may see it;
  // technicians get their own jobs instead (T035).
  const { can } = useTenantPermissions(currentTenant);
  const seesMoney = can(APP_PERMISSIONS.invoicesRead);
  // Field users (technicians) work from My day instead (T173).
  const fieldUser = !!currentTenant && isFieldUser(can);
  // "Invoice" on a completed job: the API drafts it, then it opens (T029).
  const {
    invoiceJob,
    pendingJobId: invoicingJobId,
    error: invoiceError,
  } = useInvoiceJob();
  // The owner started the onboarding wizard (T235) and hasn't finished it.
  const settings = currentTenant?.settings;
  const resumeOnboarding =
    !!user &&
    currentTenant?.owner_id === user.id &&
    (settings?.onboardingStep ?? 0) > 0 &&
    !settings?.onboardingCompletedAt;

  // Load tenants on mount to check if user has any organizations
  useEffect(() => {
    const checkTenants = async () => {
      try {
        await loadTenants();
      } catch (err) {
        console.error('Failed to load tenants:', err);
      } finally {
        setTenantsChecked(true);
      }
    };
    checkTenants();
  }, [loadTenants]);

  const loadDashboardData = useCallback(async (withMoney: boolean) => {
    setIsLoading(true);
    if (!withMoney) {
      const [jobStats, mine] = await Promise.allSettled([
        jobService.getStats(),
        jobService.getJobs({
          page: 1,
          page_size: JOBS_TO_INVOICE_SHOWN,
          sort_by: 'scheduled_start_at',
          sort_order: 'asc',
        }),
      ]);
      setData({
        ...emptyData,
        jobs: jobStats.status === 'fulfilled' ? jobStats.value : null,
        myJobs: mine.status === 'fulfilled' ? (mine.value.jobs ?? []) : null,
      });
      setIsLoading(false);
      return;
    }
    const [jobStats, invoiceStats, toInvoice, oldest, aging] =
      await Promise.allSettled([
        jobService.getStats(),
        invoiceService.getStats(),
        jobService.getJobs({
          status: JobStatus.COMPLETED,
          page: 1,
          page_size: JOBS_TO_INVOICE_SHOWN,
          sort_by: 'actual_end_at',
          sort_order: 'desc',
        }),
        jobService.getJobs({
          status: JobStatus.COMPLETED,
          page: 1,
          page_size: 1,
          sort_by: 'actual_end_at',
          sort_order: 'asc',
        }),
        arAgingService.getBucketSummary(),
      ]);

    setData({
      jobs: jobStats.status === 'fulfilled' ? jobStats.value : null,
      invoices: invoiceStats.status === 'fulfilled' ? invoiceStats.value : null,
      toInvoice:
        toInvoice.status === 'fulfilled'
          ? {
              jobs: toInvoice.value.jobs ?? [],
              total: toInvoice.value.total ?? 0,
              oldestDoneAt:
                oldest.status === 'fulfilled'
                  ? oldest.value.jobs?.[0]?.actual_end_at
                  : undefined,
            }
          : null,
      aging: aging.status === 'fulfilled' ? aging.value : null,
      myJobs: null,
    });
    setIsLoading(false);
  }, []);

  // Only load dashboard data if user has an organization
  useEffect(() => {
    if (currentTenant && !fieldUser) {
      loadDashboardData(seesMoney);
    } else {
      setIsLoading(false);
    }
  }, [currentTenant, fieldUser, seesMoney, loadDashboardData]);

  // Show loading while checking tenants
  if (!tenantsChecked || tenantsLoading) {
    return (
      <DashboardLayout>
        <div className="p-6 lg:p-8 flex items-center justify-center min-h-[400px]">
          <div className="text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
            <p className="text-neutral-600 mt-4">Loading...</p>
          </div>
        </div>
      </DashboardLayout>
    );
  }

  if (fieldUser) {
    return <Navigate to="/my-day" replace />;
  }

  // Show onboarding if user has no organization
  if (!currentTenant && tenants.length === 0) {
    return (
      <DashboardLayout>
        <div className="p-6 lg:p-8">
          <div className="max-w-2xl mx-auto text-center py-12">
            <div className="inline-flex items-center justify-center h-20 w-20 rounded-full bg-primary-100 mb-6">
              <Rocket className="h-10 w-10 text-primary-600" />
            </div>
            <h1 className="text-3xl font-bold text-neutral-900 mb-4">
              Welcome to ServicePro!
            </h1>
            <p className="text-lg text-neutral-600 mb-8">
              Get started by creating your first organization. This will be your
              workspace for managing customers, jobs, quotes, and invoices.
            </p>

            <Card variant="elevated" padding="lg" className="text-left mb-6">
              <CardContent>
                <h3 className="font-semibold text-neutral-900 mb-4">
                  What you'll be able to do:
                </h3>
                <ul className="space-y-3">
                  <li className="flex items-start">
                    <Users className="h-5 w-5 text-primary-600 mr-3 mt-0.5 shrink-0" />
                    <span className="text-neutral-700">
                      Manage your customer database and track interactions
                    </span>
                  </li>
                  <li className="flex items-start">
                    <Briefcase className="h-5 w-5 text-primary-600 mr-3 mt-0.5 shrink-0" />
                    <span className="text-neutral-700">
                      Create and schedule jobs with status tracking
                    </span>
                  </li>
                  <li className="flex items-start">
                    <FileText className="h-5 w-5 text-primary-600 mr-3 mt-0.5 shrink-0" />
                    <span className="text-neutral-700">
                      Generate professional quotes for your services
                    </span>
                  </li>
                  <li className="flex items-start">
                    <Receipt className="h-5 w-5 text-primary-600 mr-3 mt-0.5 shrink-0" />
                    <span className="text-neutral-700">
                      Send invoices and track payments
                    </span>
                  </li>
                </ul>
              </CardContent>
            </Card>

            <Button
              variant="primary"
              size="lg"
              onClick={() => navigate('/settings/organization/new')}
              className="inline-flex items-center"
            >
              <Building2 className="h-5 w-5 mr-2" />
              Create Your Organization
            </Button>
          </div>
        </div>
      </DashboardLayout>
    );
  }

  const { jobs, invoices, toInvoice, aging, myJobs } = data;
  const firstName = user?.first_name?.trim() || user?.email?.split('@')[0];
  const loadingValue = '…';

  const activeJobs =
    (jobs?.scheduled_jobs ?? 0) +
    (jobs?.in_progress_jobs ?? 0) +
    (jobs?.on_hold_jobs ?? 0);
  const jobsToInvoiceCount =
    toInvoice?.total ?? jobs?.by_status?.[JobStatus.COMPLETED] ?? 0;
  const over30 = aging ? sumBuckets(aging, OVER_30_DAYS) : null;

  let arSubline: string | undefined;
  if (over30 !== null && over30 > 0) {
    arSubline = `${formatDollars(over30, { wholeUnits: true })} over 30 days`;
  } else if (invoices && toAmount(invoices.overdue_amount) > 0) {
    arSubline = `${formatDollars(invoices.overdue_amount, { wholeUnits: true })} overdue`;
  } else if (invoices) {
    arSubline =
      invoices.outstanding_invoices > 0
        ? 'Nothing past due'
        : 'No open invoices';
  }

  const canCreateJob = can(APP_PERMISSIONS.jobsCreate);
  const quickActions = [
    {
      label: 'New quote',
      icon: FileText,
      to: '/quotes/new',
      permission: APP_PERMISSIONS.quotesCreate,
    },
    {
      label: 'Create invoice',
      icon: Receipt,
      to: '/invoices/new',
      permission: APP_PERMISSIONS.invoicesCreate,
    },
    {
      label: 'Add customer',
      icon: Building2,
      to: '/customers/new',
      permission: APP_PERMISSIONS.customersCreate,
    },
    {
      label: 'New job',
      icon: Briefcase,
      to: '/jobs/new',
      permission: APP_PERMISSIONS.jobsCreate,
    },
  ].filter((action) => can(action.permission));

  const agingTotal = aging
    ? aging.reduce((sum, b) => sum + toAmount(b.total_amount), 0)
    : 0;
  const shownJobs = toInvoice?.jobs ?? [];
  const moreToInvoice = Math.max(0, jobsToInvoiceCount - shownJobs.length);

  return (
    <DashboardLayout>
      <div className="mx-auto max-w-[1440px] p-4 sm:p-6 lg:p-8">
        <PageHeader
          title={`Welcome back${firstName ? `, ${firstName}` : ''}`}
          subtitle={
            seesMoney
              ? `${dates.today}. Here's where your jobs and money stand.`
              : `${dates.today}. Here are your jobs.`
          }
          actions={
            canCreateJob ? (
              <Button
                variant="primary"
                onClick={() => navigate('/jobs/new')}
                className="gap-2"
              >
                <Plus className="h-4 w-4" />
                New job
              </Button>
            ) : undefined
          }
        />

        {resumeOnboarding && (
          <div
            role="status"
            className="mb-6 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-primary-200 bg-primary-50 px-5 py-4"
          >
            <div>
              <p className="font-medium text-neutral-900">
                Finish setting up your workspace
              </p>
              <p className="text-sm text-neutral-600">
                Step {settings?.onboardingStep} of 3: billing defaults, tax and
                payouts.
              </p>
            </div>
            <Button variant="primary" onClick={() => navigate('/onboarding')}>
              Continue setup
            </Button>
          </div>
        )}

        {!seesMoney ? (
          <>
            <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:gap-6">
              <StatsCard
                label="Your active jobs"
                value={isLoading ? loadingValue : activeJobs}
                icon={<Wrench className="h-6 w-6" />}
                change={
                  jobs
                    ? `${jobs.scheduled_jobs} scheduled · ${jobs.in_progress_jobs} in progress`
                    : undefined
                }
              />
              <StatsCard
                label="Completed"
                value={isLoading ? loadingValue : (jobs?.completed_jobs ?? 0)}
                icon={<Briefcase className="h-6 w-6" />}
              />
            </div>
            <ListCard
              title="Your jobs"
              count={isLoading ? undefined : myJobs?.length}
              action={{ label: 'View all', to: '/jobs' }}
              bodyClassName="px-2 pb-2"
            >
              {isLoading ? (
                <p className="px-4 py-8 text-center text-sm text-neutral-500">
                  Loading…
                </p>
              ) : !myJobs || myJobs.length === 0 ? (
                <div className="px-4 py-10 text-center">
                  <p className="font-medium text-neutral-900">
                    No jobs assigned to you
                  </p>
                  <p className="mt-1 text-sm text-neutral-500">
                    Jobs the office assigns to you show up here.
                  </p>
                </div>
              ) : (
                <ul className="space-y-1">
                  {myJobs.map((job, i) => (
                    <ListCardRow
                      key={job.id}
                      className={cn(i % 2 === 0 && 'rounded-xl bg-neutral-50')}
                      to={`/jobs/${job.id}`}
                      reference={job.job_number}
                      title={customerName(job)}
                      subtitle={
                        job.scheduled_start_at
                          ? `${job.title} · ${dates.shortEpoch(job.scheduled_start_at)}`
                          : job.title
                      }
                    />
                  ))}
                </ul>
              )}
            </ListCard>
          </>
        ) : (
          <>
            {/* KPI cards */}
            <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4 lg:gap-6">
              <StatsCard
                label="Active jobs"
                value={isLoading ? loadingValue : activeJobs}
                icon={<Wrench className="h-6 w-6" />}
                change={
                  jobs
                    ? `${jobs.scheduled_jobs} scheduled · ${jobs.in_progress_jobs} in progress`
                    : undefined
                }
              />
              <StatsCard
                label="Jobs needing invoice"
                value={isLoading ? loadingValue : jobsToInvoiceCount}
                icon={<Receipt className="h-6 w-6" />}
                change={
                  toInvoice?.oldestDoneAt
                    ? `Oldest done ${dates.shortEpoch(toInvoice.oldestDoneAt)}`
                    : jobsToInvoiceCount === 0 && !isLoading
                      ? 'All caught up'
                      : undefined
                }
              />
              <StatsCard
                label="Outstanding A/R"
                value={
                  isLoading
                    ? loadingValue
                    : formatDollars(invoices?.total_outstanding ?? 0, {
                        wholeUnits: true,
                      })
                }
                icon={<Hourglass className="h-6 w-6" />}
                change={arSubline}
                changeType={
                  (over30 ?? 0) > 0 || toAmount(invoices?.overdue_amount) > 0
                    ? 'negative'
                    : 'neutral'
                }
              />
              <StatsCard
                label={`Collected in ${dates.thisMonth}`}
                value={
                  isLoading
                    ? loadingValue
                    : formatDollars(invoices?.collected_this_month ?? 0, {
                        wholeUnits: true,
                      })
                }
                icon={<HandCoins className="h-6 w-6" />}
                change={`Payments since ${dates.thisMonth} 1`}
              />
            </div>

            {invoiceError && (
              <div
                role="alert"
                className="mb-6 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700"
              >
                {invoiceError}
              </div>
            )}

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
              {/* Jobs needing invoice */}
              <ListCard
                className="lg:col-span-7"
                title="Jobs needing invoice"
                count={isLoading ? undefined : jobsToInvoiceCount}
                action={{ label: 'View all', to: '/jobs' }}
                bodyClassName="px-2 pb-2"
                footer={
                  shownJobs.length > 0 ? (
                    <>
                      <span className="text-sm text-neutral-600">
                        {moreToInvoice > 0
                          ? `${moreToInvoice} more ready to invoice`
                          : 'Each job becomes its own invoice'}
                      </span>
                      <Link
                        to="/invoices"
                        className="text-sm font-medium text-primary-600 hover:text-primary-700"
                      >
                        Go to invoices
                      </Link>
                    </>
                  ) : undefined
                }
              >
                {isLoading ? (
                  <p className="px-4 py-8 text-center text-sm text-neutral-500">
                    Loading…
                  </p>
                ) : shownJobs.length === 0 ? (
                  <div className="px-4 py-10 text-center">
                    <p className="font-medium text-neutral-900">
                      Nothing waiting to be invoiced
                    </p>
                    <p className="mt-1 text-sm text-neutral-500">
                      Completed jobs show up here until they're invoiced.
                    </p>
                  </div>
                ) : (
                  <ul className="space-y-1">
                    {shownJobs.map((job, i) => {
                      const amount = jobAmount(job);
                      const done = job.actual_end_at
                        ? `done ${dates.shortEpoch(job.actual_end_at)}`
                        : 'completed';
                      return (
                        <ListCardRow
                          key={job.id}
                          className={cn(
                            i % 2 === 0 && 'rounded-xl bg-neutral-50'
                          )}
                          to={`/jobs/${job.id}`}
                          reference={job.job_number}
                          title={customerName(job)}
                          subtitle={`${job.title} · ${done}`}
                          value={
                            amount !== undefined ? formatDollars(amount) : null
                          }
                          trailing={
                            // Agreement visits go on the consolidated invoice.
                            job.agreement_id ? (
                              <span className="mr-3 shrink-0 text-xs text-neutral-500">
                                On agreement
                              </span>
                            ) : (
                              <button
                                type="button"
                                onClick={() => invoiceJob(job.id)}
                                disabled={invoicingJobId !== null}
                                className="mr-3 shrink-0 rounded-lg border border-primary-600 px-3 py-1.5 text-sm font-medium text-primary-600 hover:bg-primary-50 disabled:opacity-60"
                                aria-label={`Create invoice for ${job.job_number}`}
                              >
                                {invoicingJobId === job.id
                                  ? 'Invoicing…'
                                  : 'Invoice'}
                              </button>
                            )
                          }
                        />
                      );
                    })}
                  </ul>
                )}
              </ListCard>

              <div className="flex flex-col gap-6 lg:col-span-5">
                {/* A/R aging */}
                <ListCard
                  title="A/R aging"
                  action={
                    <Link
                      to="/reports/ar-aging"
                      className="text-[15px] font-semibold text-neutral-900 tabular-nums hover:text-primary-700"
                      title="Open the A/R aging report"
                    >
                      {aging
                        ? formatDollars(agingTotal, { wholeUnits: true })
                        : ''}
                    </Link>
                  }
                  bodyClassName="px-6 pb-6"
                >
                  {isLoading ? (
                    <p className="py-6 text-center text-sm text-neutral-500">
                      Loading…
                    </p>
                  ) : !aging ? (
                    <p className="py-6 text-center text-sm text-neutral-500">
                      Aging isn't available right now.
                    </p>
                  ) : agingTotal <= 0 ? (
                    <p className="py-6 text-center text-sm text-neutral-500">
                      No money owed right now.
                    </p>
                  ) : (
                    <>
                      <div
                        className="mb-4 flex h-3 gap-0.5 overflow-hidden rounded-full bg-neutral-100"
                        role="img"
                        aria-label="Outstanding balance by age"
                      >
                        {AGING_BUCKETS.map((bucket) => {
                          const amount = sumBuckets(aging, [bucket]);
                          if (amount <= 0) return null;
                          return (
                            <div
                              key={bucket}
                              className={AGING_BUCKET_STYLES[bucket].bar}
                              style={{
                                width: `${(amount / agingTotal) * 100}%`,
                              }}
                            />
                          );
                        })}
                      </div>
                      <ul className="space-y-2">
                        {AGING_BUCKETS.map((bucket) => (
                          <li
                            key={bucket}
                            className="flex items-center justify-between text-[15px]"
                          >
                            <span className="flex items-center gap-2.5 text-neutral-700">
                              <span
                                className={cn(
                                  'h-2.5 w-2.5 rounded-full',
                                  AGING_BUCKET_STYLES[bucket].bar
                                )}
                              />
                              {AGING_BUCKET_STYLES[bucket].shortLabel}
                            </span>
                            <span className="font-medium text-neutral-900 tabular-nums">
                              {formatDollars(sumBuckets(aging, [bucket]), {
                                wholeUnits: true,
                              })}
                            </span>
                          </li>
                        ))}
                      </ul>
                    </>
                  )}
                </ListCard>

                {/* Quick actions */}
                {quickActions.length > 0 && (
                  <ListCard title="Quick actions" bodyClassName="px-6 pb-6">
                    <div className="grid grid-cols-2 gap-3">
                      {quickActions.map((action) => {
                        const Icon = action.icon;
                        return (
                          <Link
                            key={action.label}
                            to={action.to}
                            className="flex items-center gap-3 rounded-xl border border-neutral-200 px-3 py-3 text-[15px] text-neutral-800 transition-colors hover:border-primary-300 hover:bg-primary-50"
                          >
                            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600">
                              <Icon className="h-4 w-4" />
                            </span>
                            <span className="truncate">{action.label}</span>
                          </Link>
                        );
                      })}
                    </div>
                  </ListCard>
                )}
              </div>
            </div>
          </>
        )}
      </div>
    </DashboardLayout>
  );
}
